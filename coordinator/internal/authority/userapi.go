// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/coordinator/internal/seedengine"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/crypto"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// SetManifest registers a new manifest at the Coordinator.
func (a *Authority) SetManifest(ctx context.Context, req *userapi.SetManifestRequest) (*userapi.SetManifestResponse, error) {
	a.logger.Info("SetManifest called")

	if err := a.syncState(); errors.Is(err, ErrNeedsRecovery) {
		return nil, status.Error(codes.FailedPrecondition, ErrNeedsRecovery.Error())
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "syncing internal state: %v", err)
	}

	var m *manifest.Manifest
	if err := json.Unmarshal(req.Manifest, &m); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unmarshaling manifest: %v", err)
	}

	var resp userapi.SetManifestResponse

	oldState := a.state.Load()
	var se *seedengine.SeedEngine
	if oldState != nil {
		oldManifest := oldState.Manifest()
		// Subsequent SetManifest call, check permissions of caller.
		if err := a.validatePeer(ctx, oldManifest.WorkloadOwnerKeyDigests); err != nil {
			a.logger.Warn("SetManifest peer validation failed", "err", err)
			return nil, status.Errorf(codes.PermissionDenied, "validating peer: %v", err)
		}
		se = oldState.SeedEngine()
		if slices.Compare(oldManifest.SeedshareOwnerPubKeys, m.SeedshareOwnerPubKeys) != 0 {
			a.logger.Warn("SetManifest detected attempted seedshare owners change", "from", oldManifest.SeedshareOwnerPubKeys, "to", m.SeedshareOwnerPubKeys)
			return nil, status.Errorf(codes.PermissionDenied, "changes to seedshare owners are not allowed")
		}
	} else {
		// First SetManifest call, initialize seed engine.
		seed, err := crypto.GenerateRandomBytes(constants.SecretSeedSize)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generating random bytes for seed: %v", err)
		}
		salt, err := crypto.GenerateRandomBytes(constants.SecretSeedSaltSize)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generating random bytes for seed salt: %v", err)
		}

		seedShares, err := manifest.EncryptSeedShares(seed, m.SeedshareOwnerPubKeys)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "initializing seed engine: %v", err)
		}
		se, err = seedengine.New(seed, salt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "setting seed: %v", err)
		}
		resp.SeedSharesDoc = &userapi.SeedShareDocument{
			Salt:       salt,
			SeedShares: seedShares,
		}
	}

	// Store resources in History.

	policyMap := make(map[[history.HashSize]byte][]byte)
	for _, policy := range req.GetPolicies() {
		policyHash, err := a.hist.SetPolicy(policy)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "setting policy: %v", err)
		}
		policyMap[policyHash] = policy
	}

	for hexRef := range m.Policies {
		var ref [history.HashSize]byte
		refSlice, err := hexRef.Bytes()
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid policy hash: %v", err)
		}
		copy(ref[:], refSlice)
		if _, ok := policyMap[ref]; !ok {
			return nil, status.Errorf(codes.InvalidArgument, "no policy provided for hash %q", hexRef)
		}
	}

	manifestHash, err := a.hist.SetManifest(req.GetManifest())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "setting manifest: %v", err)
	}

	// Advance state.

	nextTransition := &history.Transition{
		ManifestHash: manifestHash,
	}
	var oldLatest *history.LatestTransition
	var oldGeneration int
	if oldState != nil {
		nextTransition.PreviousTransitionHash = oldState.latest.TransitionHash
		oldLatest = oldState.latest
		oldGeneration = oldState.generation
	}
	nextTransitionHash, err := a.hist.SetTransition(nextTransition)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "setting transition: %v", err)
	}
	nextLatest := &history.LatestTransition{TransitionHash: nextTransitionHash}

	if err := a.hist.SetLatest(oldLatest, nextLatest, se.TransactionSigningKey()); err != nil {
		return nil, status.Errorf(codes.Internal, "setting latest: %v", err)
	}

	meshKey, err := se.GenerateMeshCAKey()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "deriving mesh CA key: %v", err)
	}
	ca, err := ca.New(se.RootCAKey(), meshKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "creating CA: %v", err)
	}

	nextState := &State{
		seedEngine:    se,
		latest:        nextLatest,
		manifest:      m,
		manifestBytes: req.GetManifest(),
		ca:            ca,
		generation:    oldGeneration + 1,
	}

	if a.state.CompareAndSwap(oldState, nextState) {
		a.metrics.manifestGeneration.Set(float64(nextState.generation))
	}
	// If the CompareAndSwap did not go through, this means that another SetManifest happened in
	// the meantime. This is fine: we know that m.state must be a transition after ours because
	// the SetLatest call succeeded. That other SetManifest call must have been operating on our
	// nextState already, because it had to refer to our transition. Thus, we can forget about
	// the state, except that we need to return the right CA for the manifest _our_ user set.

	resp.RootCA = ca.GetRootCACert()
	resp.MeshCA = ca.GetMeshCACert()

	a.logger.Info("SetManifest succeeded")
	return &resp, nil
}

// GetManifests retrieves the current CA certificates, the manifest history and all policies.
func (a *Authority) GetManifests(_ context.Context, _ *userapi.GetManifestsRequest,
) (*userapi.GetManifestsResponse, error) {
	a.logger.Info("GetManifest called")
	if err := a.syncState(); err != nil {
		return nil, status.Errorf(codes.Internal, "syncing internal state: %v", err)
	}
	state := a.state.Load()
	if state == nil {
		return nil, status.Error(codes.FailedPrecondition, ErrNoManifest.Error())
	}

	var manifests [][]byte
	policies := make(map[manifest.HexString][]byte)
	err := a.walkTransitions(state.latest.TransitionHash, func(_ [history.HashSize]byte, t *history.Transition) error {
		manifestBytes, err := a.hist.GetManifest(t.ManifestHash)
		if err != nil {
			return err
		}
		manifests = append(manifests, manifestBytes)

		var mnfst manifest.Manifest
		if err := json.Unmarshal(manifestBytes, &mnfst); err != nil {
			return err
		}

		for policyHashHex := range mnfst.Policies {
			policyHash, err := policyHashHex.Bytes()
			if err != nil {
				return fmt.Errorf("converting hex to bytes: %w", err)
			}
			var policyHashFixed [history.HashSize]byte
			copy(policyHashFixed[:], policyHash)
			policyBytes, err := a.hist.GetPolicy(policyHashFixed)
			if err != nil {
				return fmt.Errorf("getting policy: %w", err)
			}
			policies[policyHashHex] = policyBytes
		}
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetching manifests from history: %v", err)
	}
	// Traversing the history yields manifests in the wrong order, so reverse the slice.
	slices.Reverse(manifests)

	ca := state.CA()
	resp := &userapi.GetManifestsResponse{
		Manifests: manifests,
		RootCA:    ca.GetRootCACert(),
		MeshCA:    ca.GetMeshCACert(),
	}
	for _, policy := range policies {
		resp.Policies = append(resp.Policies, policy)
	}

	a.logger.Info("GetManifest succeeded")
	return resp, nil
}

// Recover recovers the Coordinator from a seed and salt.
func (a *Authority) Recover(ctx context.Context, req *userapi.RecoverRequest) (*userapi.RecoverResponse, error) {
	a.logger.Info("Recover called")

	if a.state.Load() != nil {
		return nil, status.Error(codes.FailedPrecondition, ErrAlreadyRecovered.Error())
	}

	hasLatest, err := a.hist.HasLatest()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "accessing history: %v", err)
	}
	if !hasLatest {
		return nil, status.Errorf(codes.FailedPrecondition, "no persisted transaction to recover from")
	}

	se, err := seedengine.New(req.Seed, req.Salt)
	if err != nil {
		// Pretty sure this failed because the seed was bad.
		return nil, status.Errorf(codes.InvalidArgument, "initializing seed engine: %v", err)
	}

	state, err := a.fetchState(se)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "recovery failed: %v", err)
	}

	var digests []manifest.HexString
	for _, pubKey := range state.Manifest().SeedshareOwnerPubKeys {
		bytes, err := pubKey.Bytes()
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "seedshare owner public key is not hex-encoded")
		}
		sum := sha256.Sum256(bytes)
		digests = append(digests, manifest.NewHexString(sum[:]))
	}
	if err := a.validatePeer(ctx, digests); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "peer not authorized to recover existing state: %v", err)
	}

	if !a.state.CompareAndSwap(nil, state) {
		return nil, status.Errorf(codes.FailedPrecondition, "the coordinator was recovered concurrently")
	}
	return &userapi.RecoverResponse{}, nil
}

func (a *Authority) validatePeer(ctx context.Context, keyDigests []manifest.HexString) error {
	if len(keyDigests) == 0 {
		return errors.New("setting manifest is disabled")
	}

	peerPubKey, err := getPeerPublicKey(ctx)
	if err != nil {
		return err
	}
	peerPub256Sum := sha256.Sum256(peerPubKey)
	for _, key := range keyDigests {
		trustedWorkloadOwnerSHA256, err := key.Bytes()
		if err != nil {
			return fmt.Errorf("parsing key: %w", err)
		}
		if bytes.Equal(peerPub256Sum[:], trustedWorkloadOwnerSHA256) {
			return nil
		}
	}
	return errors.New("peer not authorized")
}

func getPeerPublicKey(ctx context.Context) ([]byte, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("no peer found in context")
	}
	tlsInfo, ok := peer.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, fmt.Errorf("peer auth info is not of type TLSInfo: got %T", peer.AuthInfo)
	}
	if len(tlsInfo.State.PeerCertificates) == 0 || tlsInfo.State.PeerCertificates[0] == nil {
		return nil, errors.New("no peer certificates found")
	}

	switch pubKey := tlsInfo.State.PeerCertificates[0].PublicKey.(type) {
	case *rsa.PublicKey:
		return x509.MarshalPKCS1PublicKey(pubKey), nil
	case *ecdsa.PublicKey:
		return x509.MarshalPKIXPublicKey(pubKey)
	default:
		return nil, fmt.Errorf("unsupported peer public key type %T", pubKey)
	}
}

var (
	// ErrAlreadyRecovered is returned if seedEngine initialization was requested but a seed is already set.
	ErrAlreadyRecovered = errors.New("coordinator is already recovered")
	// ErrNeedsRecovery is returned if state exists, but no secrets are available, e.g. after restart.
	ErrNeedsRecovery = errors.New("coordinator is in recovery mode")
)
