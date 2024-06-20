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

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/internal/ca"
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

	if err := a.syncState(); err != nil {
		return nil, status.Errorf(codes.Internal, "syncing internal state: %v", err)
	}

	var m *manifest.Manifest
	if err := json.Unmarshal(req.Manifest, &m); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unmarshaling manifest: %v", err)
	}

	var resp userapi.SetManifestResponse

	oldState := a.state.Load()
	if oldState == nil {
		// First SetManifest call, initialize seed engine.
		seedSalt, err := crypto.GenerateRandomBytes(64)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generating random bytes: %v", err)
		}
		seed, salt := seedSalt[:32], seedSalt[32:]

		// TODO(burgerdev): requires https://github.com/edgelesssys/contrast/commit/bae8e7f
		seedShares, err := manifest.EncryptSeedShares(seed /*m.SeedshareOwnerPubKeys*/, nil)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "initializing seed engine: %v", err)
		}
		if err := a.initSeedEngine(seed, salt); err != nil {
			return nil, status.Errorf(codes.Internal, "setting seed: %v", err)
		}
		resp.SeedSharesDoc = &userapi.SeedShareDocument{
			Salt:       salt,
			SeedShares: seedShares,
		}
	} else {
		// Subsequent SetManifest call, check permissions of caller.
		if err := a.validatePeer(ctx, oldState.manifest); err != nil {
			a.logger.Warn("SetManifest peer validation failed", "err", err)
			return nil, status.Errorf(codes.PermissionDenied, "validating peer: %v", err)
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
	meshKey, err := se.DeriveMeshCAKey(nextTransitionHash)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "deriving mesh CA key: %v", err)
	}
	ca, err := ca.New(se.RootCAKey(), meshKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "creating CA: %v", err)
	}

	nextState := &state{
		latest:     nextLatest,
		manifest:   m,
		ca:         ca,
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

	manifests, ca, err := a.getManifestsAndLatestCA()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting manifests: %v", err)
	}
	if len(manifests) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "no manifests set")
	}

	manifestBytes, err := manifestSliceToBytesSlice(manifests)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshaling manifests: %v", err)
	}

	// TODO(burgerdev): consolidate with getManifestsAndLatestCA
	policies := make(map[manifest.HexString][]byte)
	err = a.walkTransitions(a.state.Load().latest.TransitionHash, func(_ [history.HashSize]byte, t *history.Transition) error {
		manifestBytes, err := a.hist.GetManifest(t.ManifestHash)
		if err != nil {
			return fmt.Errorf("getting manifest: %w", err)
		}
		var mnfst manifest.Manifest
		if err := json.Unmarshal(manifestBytes, &mnfst); err != nil {
			return fmt.Errorf("decoding manifest: %w", err)
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
		return nil, status.Errorf(codes.Internal, "querying policies: %v", err)
	}

	resp := &userapi.GetManifestsResponse{
		Manifests: manifestBytes,
		RootCA:    ca.GetRootCACert(),
		MeshCA:    ca.GetMeshCACert(),
	}
	for _, policy := range policies {
		resp.Policies = append(resp.Policies, policy)
	}

	a.logger.Info("GetManifest succeeded")
	return resp, nil
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

func manifestSliceToBytesSlice(s []*manifest.Manifest) ([][]byte, error) {
	var manifests [][]byte
	for i, manifest := range s {
		manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("mashaling manifest %d manifest: %w", i, err)
		}
		manifests = append(manifests, manifestBytes)
	}
	return manifests, nil
}
