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

	if err := a.validatePeer(ctx); err != nil {
		a.logger.Warn("SetManifest peer validation failed", "err", err)
		return nil, status.Errorf(codes.PermissionDenied, "validating peer: %v", err)
	}

	var m *manifest.Manifest
	if err := json.Unmarshal(req.Manifest, &m); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unmarshaling manifest: %v", err)
	}

	if len(m.Policies) != len(req.Policies) {
		return nil, status.Error(codes.InvalidArgument, "request must contain exactly the policies referenced in the manifest")
	}

	ca, err := a.setManifest(req.GetManifest(), req.GetPolicies())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "setting manifest: %v", err)
	}

	resp := &userapi.SetManifestResponse{
		RootCA: ca.GetRootCACert(),
		MeshCA: ca.GetMeshCACert(),
	}

	a.logger.Info("SetManifest succeeded")
	return resp, nil
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

func (a *Authority) validatePeer(ctx context.Context) error {
	latest, err := a.latestManifest()
	if err != nil && errors.Is(err, ErrNoManifest) {
		// in the initial state, no peer validation is required
		return nil
	}
	if err != nil {
		return fmt.Errorf("getting latest manifest: %w", err)
	}
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
