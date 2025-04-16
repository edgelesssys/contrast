// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// Package userapi implements the userapi.UserAPI gRPC service.
//
// It is responsible for
//   * translating between RPCs and library calls
//   * authorizing users
//   * managing content-addressed history elements (but not the LatestTransition)
//   * creating secrets (seed and mesh CA key)
//
// The package has access to an authority.Authority, which is the source of truth for the current
// state of the system, and a history.History for storing supporting content (manifests and
// policies).
package userapi

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
	"log/slog"
	"slices"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/coordinator/internal/authority"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/crypto"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/seedengine"
	"github.com/edgelesssys/contrast/internal/userapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type Authority interface {
	GetState() (*authority.State, error)
	UpdateState(*authority.State, *seedengine.SeedEngine, []byte, *ecdsa.PrivateKey) (*authority.State, error)
	ResetState(*authority.State, *seedengine.SeedEngine, *history.LatestTransition, *ecdsa.PrivateKey) (*authority.State, error)
}

type Server struct {
	logger *slog.Logger
	hist   *history.History
	auth   Authority

	userapi.UnimplementedUserAPIServer
}

func New(logger *slog.Logger, hist *history.History, auth Authority) *Server {
	return &Server{
		logger: logger,
		hist:   hist,
		auth:   auth,
	}
}

// SetManifest registers a new manifest at the Coordinator.
func (s *Server) SetManifest(ctx context.Context, req *userapi.SetManifestRequest) (*userapi.SetManifestResponse, error) {
	s.logger.Info("SetManifest called")

	oldState, err := s.auth.GetState()
	if err != nil && !errors.Is(err, authority.ErrNoState) {
		return nil, status.Errorf(codes.FailedPrecondition, "getting state: %v", err)
	}
	if oldState == nil {
		hasLatest, err := s.hist.HasLatest()
		if err != nil {
			return nil, fmt.Errorf("checking latest state: %w", err)
		}
		if hasLatest {
			return nil, status.Errorf(codes.FailedPrecondition, authority.ErrNeedsRecovery.Error())
		}
	} else if oldState.IsStale() {
		return nil, status.Errorf(codes.FailedPrecondition, authority.ErrNeedsRecovery.Error())
	}

	var m *manifest.Manifest
	if err := json.Unmarshal(req.Manifest, &m); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unmarshaling manifest: %v", err)
	}

	var resp userapi.SetManifestResponse

	var se *seedengine.SeedEngine
	if oldState != nil {
		oldManifest := oldState.Manifest()
		// Subsequent SetManifest call, check permissions of caller.
		if err := validatePeer(ctx, oldManifest.WorkloadOwnerKeyDigests); err != nil {
			s.logger.Warn("SetManifest peer validation failed", "err", err)
			return nil, status.Errorf(codes.PermissionDenied, "validating peer: %v", err)
		}
		se = oldState.SeedEngine()
		if slices.Compare(oldManifest.SeedshareOwnerPubKeys, m.SeedshareOwnerPubKeys) != 0 {
			s.logger.Warn("SetManifest detected attempted seedshare owners change", "from", oldManifest.SeedshareOwnerPubKeys, "to", m.SeedshareOwnerPubKeys)
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
		policyHash, err := s.hist.SetPolicy(policy)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "storing policy: %v", err)
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

	meshKey, err := se.GenerateMeshCAKey()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "deriving mesh CA key: %v", err)
	}
	state, err := s.auth.UpdateState(oldState, se, req.Manifest, meshKey)
	if err != nil {
		code := codes.Internal
		if errors.Is(err, authority.ErrConcurrentUpdate) {
			code = codes.FailedPrecondition
		}
		return nil, status.Errorf(code, "updating Coordinator state: %v", err)
	}
	resp.MeshCA = state.CA().GetMeshCACert()
	resp.RootCA = state.CA().GetRootCACert()

	s.logger.Info("SetManifest succeeded")
	return &resp, nil
}

// GetManifests retrieves the current CA certificates, the manifest history and all policies.
func (s *Server) GetManifests(_ context.Context, _ *userapi.GetManifestsRequest,
) (*userapi.GetManifestsResponse, error) {
	s.logger.Info("GetManifest called")
	state, err := s.auth.GetState()
	if err != nil {
		if errors.Is(err, authority.ErrNoState) {
			hasLatest, err := s.hist.HasLatest()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not read store: %v", err)
			}
			if hasLatest {
				return nil, status.Error(codes.FailedPrecondition, authority.ErrNeedsRecovery.Error())
			}
			return nil, status.Error(codes.FailedPrecondition, authority.ErrNoManifest.Error())
		}
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}
	if state.IsStale() {
		return nil, status.Error(codes.FailedPrecondition, authority.ErrNeedsRecovery.Error())
	}

	var manifests [][]byte
	policies := make(map[manifest.HexString][]byte)
	latestTransitionHash := state.LatestTransitionHash()
	err = s.hist.WalkTransitions(latestTransitionHash, func(_ [history.HashSize]byte, t *history.Transition) error {
		manifestBytes, err := s.hist.GetManifest(t.ManifestHash)
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
			policyBytes, err := s.hist.GetPolicy(policyHashFixed)
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

	s.logger.Info("GetManifest succeeded")
	return resp, nil
}

// Recover recovers the Coordinator from a seed and salt.
func (s *Server) Recover(ctx context.Context, req *userapi.RecoverRequest) (*userapi.RecoverResponse, error) {
	s.logger.Info("Recover called")

	oldState, err := s.auth.GetState()
	if err != nil && !errors.Is(err, authority.ErrNoState) {
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}

	if oldState != nil && !oldState.IsStale() {
		return nil, status.Errorf(codes.FailedPrecondition, ErrAlreadyRecovered.Error())
	}

	hasLatest, err := s.hist.HasLatest()
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

	insecureLatest, err := s.hist.GetLatestInsecure()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Getting latest transition: %v", err)
	}
	transition, err := s.hist.GetTransition(insecureLatest.TransitionHash)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Getting transition: %v", err)
	}
	manifestBytes, err := s.hist.GetManifest(transition.ManifestHash)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Getting manifest: %v", err)
	}
	var mnfst manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &mnfst); err != nil {
		return nil, status.Errorf(codes.Internal, "Unmarshaling latest manifest: %v", err)
	}

	var digests []manifest.HexString
	for _, pubKey := range mnfst.SeedshareOwnerPubKeys {
		bytes, err := pubKey.Bytes()
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "seedshare owner public key is not hex-encoded")
		}
		sum := sha256.Sum256(bytes)
		digests = append(digests, manifest.NewHexString(sum[:]))
	}
	if err := validatePeer(ctx, digests); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "peer not authorized to recover existing state: %v", err)
	}

	latest, err := s.hist.GetLatest(&se.TransactionSigningKey().PublicKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Reading latest transition with new seedengine: %v", err)
	}
	if latest.TransitionHash != insecureLatest.TransitionHash {
		return nil, status.Errorf(codes.FailedPrecondition, "latest transition changed from %x to %x", insecureLatest.TransitionHash, latest.TransitionHash)
	}

	meshKey, err := se.GenerateMeshCAKey()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "deriving mesh CA key: %v", err)
	}
	_, err = s.auth.ResetState(oldState, se, latest, meshKey)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Could not reset internal state: %v", err)
	}
	return &userapi.RecoverResponse{}, nil
}

func validatePeer(ctx context.Context, keyDigests []manifest.HexString) error {
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

// ErrAlreadyRecovered is returned if seedEngine initialization was requested but a seed is already set.
var ErrAlreadyRecovered = errors.New("coordinator is already recovered")
