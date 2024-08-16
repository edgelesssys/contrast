// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/edgelesssys/contrast/coordinator/history"
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
	if oldState != nil {
		// Subsequent SetManifest call, check permissions of caller.
		if err := a.validatePeer(ctx, oldState.Manifest); err != nil {
			a.logger.Warn("SetManifest peer validation failed", "err", err)
			return nil, status.Errorf(codes.PermissionDenied, "validating peer: %v", err)
		}
	} else if a.se.Load() == nil {
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
		if err := a.initSeedEngine(seed, salt); errors.Is(err, ErrAlreadyRecovered) {
			return nil, status.Error(codes.Unavailable, "concurrent initialization through SetManifest detected")
		} else if err != nil {
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

	if err := a.hist.SetLatest(oldLatest, nextLatest); err != nil {
		return nil, status.Errorf(codes.Internal, "setting latest: %v", err)
	}

	se := a.se.Load()
	meshKey, err := se.GenerateMeshCAKey()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "deriving mesh CA key: %v", err)
	}
	ca, err := ca.New(se.RootCAKey(), meshKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "creating CA: %v", err)
	}

	nextState := &State{
		latest:     nextLatest,
		Manifest:   m,
		CA:         ca,
		generation: oldGeneration + 1,
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

	resp := &userapi.GetManifestsResponse{
		Manifests: manifests,
		RootCA:    state.CA.GetRootCACert(),
		MeshCA:    state.CA.GetMeshCACert(),
	}
	for _, policy := range policies {
		resp.Policies = append(resp.Policies, policy)
	}

	a.logger.Info("GetManifest succeeded")
	return resp, nil
}

// Recover recovers the Coordinator from a seed and salt.
func (a *Authority) Recover(_ context.Context, req *userapi.RecoverRequest) (*userapi.RecoverResponse, error) {
	a.logger.Info("Recover called")

	if err := a.initSeedEngine(req.Seed, req.Salt); errors.Is(err, ErrAlreadyRecovered) {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	} else if err != nil {
		// Pretty sure this failed because the seed was bad.
		return nil, status.Errorf(codes.InvalidArgument, "initializing seed engine: %v", err)
	}

	if err := a.syncState(); err != nil {
		// This recovery attempt did not lead to a good state, let's roll it back.
		a.se.Store(nil)
		return nil, status.Errorf(codes.InvalidArgument, "recovery failed and was rolled back: %v", err)
	}
	return &userapi.RecoverResponse{}, nil
}

func (a *Authority) validatePeer(ctx context.Context, latest *manifest.Manifest) error {
	if len(latest.WorkloadOwnerKeyDigests) == 0 {
		return errors.New("setting manifest is disabled")
	}

	peerPubKey, err := getPeerPublicKey(ctx)
	if err != nil {
		return err
	}
	peerPub256Sum := sha256.Sum256(peerPubKey)
	for _, key := range latest.WorkloadOwnerKeyDigests {
		trustedWorkloadOwnerSHA256, err := key.Bytes()
		if err != nil {
			return fmt.Errorf("parsing key: %w", err)
		}
		if bytes.Equal(peerPub256Sum[:], trustedWorkloadOwnerSHA256) {
			return nil
		}
	}
	return errors.New("peer not authorized workload owner")
}

func getPeerPublicKey(ctx context.Context) ([]byte, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("no peer found in context")
	}
	tlsInfo, ok := peer.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, errors.New("peer auth info is not of type TLSInfo")
	}
	if len(tlsInfo.State.PeerCertificates) == 0 || tlsInfo.State.PeerCertificates[0] == nil {
		return nil, errors.New("no peer certificates found")
	}
	if tlsInfo.State.PeerCertificates[0].PublicKeyAlgorithm != x509.ECDSA {
		return nil, errors.New("peer public key is not of type ECDSA")
	}
	return x509.MarshalPKIXPublicKey(tlsInfo.State.PeerCertificates[0].PublicKey)
}

var (
	// ErrAlreadyRecovered is returned if seedEngine initialization was requested but a seed is already set.
	ErrAlreadyRecovered = errors.New("coordinator is already recovered")
	// ErrNeedsRecovery is returned if state exists, but no secrets are available, e.g. after restart.
	ErrNeedsRecovery = errors.New("coordinator is in recovery mode")
)
