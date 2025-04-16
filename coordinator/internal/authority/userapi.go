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
	"log/slog"
	"slices"

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

// Server serves the userapi.UserAPI. Servers need to be constructed with NewUserAPI.
type Server struct {
	logger *slog.Logger
	auth   *Authority

	userapi.UnimplementedUserAPIServer
}

// NewUserAPI constructs a new Server instance.
func NewUserAPI(logger *slog.Logger, auth *Authority) *Server {
	return &Server{
		logger: logger,
		auth:   auth,
	}
}

// SetManifest registers a new manifest at the Coordinator.
func (s *Server) SetManifest(ctx context.Context, req *userapi.SetManifestRequest) (*userapi.SetManifestResponse, error) {
	s.logger.Info("SetManifest called")

	oldState, err := s.auth.GetState()
	switch {
	case errors.Is(err, ErrStaleState):
		return nil, status.Error(codes.FailedPrecondition, ErrNeedsRecovery.Error())
	case errors.Is(err, ErrNoState):
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

	state, err := s.auth.UpdateState(oldState, se, req.GetManifest(), req.GetPolicies())
	if err != nil {
		code := codes.Internal
		if errors.Is(err, ErrConcurrentUpdate) {
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
	switch {
	case errors.Is(err, ErrNoState):
		return nil, status.Error(codes.FailedPrecondition, ErrNoManifest.Error())
	case errors.Is(err, ErrStaleState):
		return nil, status.Error(codes.FailedPrecondition, ErrNeedsRecovery.Error())
	case err != nil:
		return nil, status.Errorf(codes.Internal, "getting state: %v", err)
	}

	manifests, policies, err := s.auth.GetHistory()
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

	oldState, err := s.auth.GetState()
	switch {
	case errors.Is(err, ErrStaleState):
		// This is fine, we want to recover anyway.
	case errors.Is(err, ErrNoState):
		return nil, status.Error(codes.FailedPrecondition, "no state to recover from")
	case err != nil:
		return nil, status.Errorf(codes.Internal, "getting state: %v", err)
	default:
		return nil, status.Error(codes.FailedPrecondition, ErrAlreadyRecovered.Error())
	}

	_, err = s.auth.ResetState(oldState, &seedAuthorizer{ctx: ctx, req: req})
	if err != nil {
		return nil, fmt.Errorf("resetting state: %w", err)
	}
	return &userapi.RecoverResponse{}, nil
}

type seedAuthorizer struct {
	ctx context.Context
	req *userapi.RecoverRequest
}

func (a *seedAuthorizer) AuthorizeByManifest(mnfst *manifest.Manifest) (*seedengine.SeedEngine, *ecdsa.PrivateKey, error) {
	var digests []manifest.HexString
	for _, pubKey := range mnfst.SeedshareOwnerPubKeys {
		bytes, err := pubKey.Bytes()
		if err != nil {
			return nil, nil, status.Errorf(codes.FailedPrecondition, "seedshare owner public key is not hex-encoded")
		}
		sum := sha256.Sum256(bytes)
		digests = append(digests, manifest.NewHexString(sum[:]))
	}
	if err := validatePeer(a.ctx, digests); err != nil {
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
