// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package userapi implements the userapi.UserAPI gRPC service.
//
// It is responsible for
//   - translating between RPCs and library calls
//   - authorizing users
//   - creating secrets (seed and mesh CA key)
//
// The package has access to an authority.Authority, which is the source of truth for the current
// state of the system.
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

	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
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

// guard is the public API of stateguard.Guard.
type guard interface {
	// GetState returns the current state. If the error is nil, the state must be set.
	GetState(context.Context) (*stateguard.State, error)
	// GetHistory returns a slice of manifests and a map of policies referenced in the manifests.
	GetHistory(context.Context) (manifests [][]byte, policies map[manifest.HexString][]byte, err error)
	// UpdateState advances the state to the given manifest and policies.
	UpdateState(ctx context.Context, oldState *stateguard.State, se *seedengine.SeedEngine, manifest []byte, policies [][]byte) (newState *stateguard.State, err error)
	// ResetState recovers to the latest persisted state, authorizing the recovery seed with the passed func.
	ResetState(ctx context.Context, oldState *stateguard.State, a stateguard.SecretSourceAuthorizer) (newState *stateguard.State, err error)
}

type discovery interface {
	GetPeers(ctx context.Context) ([]string, error)
}

// Server serves the userapi.UserAPI. Servers need to be constructed with New.
type Server struct {
	logger    *slog.Logger
	guard     guard
	discovery discovery

	userapi.UnimplementedUserAPIServer
}

// New constructs a new Server instance.
func New(logger *slog.Logger, guard guard, discovery discovery) *Server {
	return &Server{
		logger:    logger,
		guard:     guard,
		discovery: discovery,
	}
}

// SetManifest registers a new manifest at the Coordinator.
func (s *Server) SetManifest(ctx context.Context, req *userapi.SetManifestRequest) (*userapi.SetManifestResponse, error) {
	s.logger.Info("SetManifest called")

	oldState, err := s.guard.GetState(ctx)
	switch {
	case errors.Is(err, stateguard.ErrStaleState):
		return nil, status.Error(codes.FailedPrecondition, ErrNeedsRecovery.Error())
	case errors.Is(err, stateguard.ErrNoState):
		// This is fine, we are going to set the initial manifest.
	case err != nil:
		return nil, status.Errorf(codes.Internal, "getting state: %v", err)
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

	state, err := s.guard.UpdateState(ctx, oldState, se, req.GetManifest(), req.GetPolicies())
	if err != nil {
		code := codes.Internal
		if errors.Is(err, stateguard.ErrConcurrentUpdate) {
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
func (s *Server) GetManifests(ctx context.Context, _ *userapi.GetManifestsRequest) (*userapi.GetManifestsResponse, error) {
	s.logger.Info("GetManifest called")
	state, err := s.guard.GetState(ctx)
	switch {
	case errors.Is(err, stateguard.ErrNoState):
		return nil, status.Error(codes.FailedPrecondition, ErrNoManifest.Error())
	case errors.Is(err, stateguard.ErrStaleState):
		return nil, status.Error(codes.FailedPrecondition, ErrNeedsRecovery.Error())
	case err != nil:
		return nil, status.Errorf(codes.Internal, "getting state: %v", err)
	}

	manifests, policies, err := s.guard.GetHistory(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting history: %v", err)
	}

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

	// Check whether recovery is needed.
	oldState, err := s.guard.GetState(ctx)
	switch {
	case errors.Is(err, stateguard.ErrStaleState):
		// This is fine, we want to recover anyway.
	case errors.Is(err, stateguard.ErrNoState):
		return nil, status.Error(codes.FailedPrecondition, "no state to recover from")
	case err != nil:
		return nil, status.Errorf(codes.Internal, "getting state: %v", err)
	default:
		return nil, status.Error(codes.FailedPrecondition, ErrAlreadyRecovered.Error())
	}

	if !req.Force {
		// Unless forced, check whether recovery is a good idea.
		peers, err := s.discovery.GetPeers(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "listing peers: %v", err)
		}
		if len(peers) > 0 {
			return nil, status.Errorf(codes.FailedPrecondition, "rejecting user recovery because %d recovered peers are available", len(peers))
		}
	} else {
		s.logger.Info("Skipping sanity checks because user recovery was forced")
	}

	_, err = s.guard.ResetState(ctx, oldState, &seedAuthorizer{req: req})
	if err != nil {
		return nil, fmt.Errorf("resetting state: %w", err)
	}
	return &userapi.RecoverResponse{}, nil
}

type seedAuthorizer struct {
	req *userapi.RecoverRequest
}

func (a *seedAuthorizer) AuthorizeByManifest(ctx context.Context, mnfst *manifest.Manifest) (*seedengine.SeedEngine, *ecdsa.PrivateKey, error) {
	var digests []manifest.HexString
	for _, pubKey := range mnfst.SeedshareOwnerPubKeys {
		bytes, err := pubKey.Bytes()
		if err != nil {
			return nil, nil, status.Errorf(codes.FailedPrecondition, "seedshare owner public key is not hex-encoded")
		}
		sum := sha256.Sum256(bytes)
		digests = append(digests, manifest.NewHexString(sum[:]))
	}
	if err := validatePeer(ctx, digests); err != nil {
		return nil, nil, status.Errorf(codes.PermissionDenied, "peer not authorized to recover existing state: %v", err)
	}

	se, err := seedengine.New(a.req.Seed, a.req.Salt)
	if err != nil {
		// Pretty sure this failed because the seed was bad.
		return nil, nil, status.Errorf(codes.InvalidArgument, "initializing seed engine: %v", err)
	}

	meshKey, err := se.GenerateMeshCAKey()
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "deriving mesh CA key: %v", err)
	}
	return se, meshKey, nil
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

var (
	// ErrNoManifest is returned when a manifest is needed but not present.
	ErrNoManifest = errors.New("no manifest configured")
	// ErrAlreadyRecovered is returned if seedEngine initialization was requested but a seed is already set.
	ErrAlreadyRecovered = errors.New("coordinator is already recovered")
	// ErrNeedsRecovery is returned if state exists, but no secrets are available, e.g. after restart.
	ErrNeedsRecovery = errors.New("coordinator is in recovery mode")
)
