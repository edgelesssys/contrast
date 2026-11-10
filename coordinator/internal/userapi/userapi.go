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
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/cryptohelpers"
	"github.com/edgelesssys/contrast/internal/history"
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

	// allowInsecure selects the manifest security level accepted by the Coordinator. It defaults to
	// secure manifests and can be switched to insecure manifests via MakeInsecure.
	allowInsecure bool

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

// MakeInsecure configures the Server to accept only manifests that contain insecure platforms.
//
// This voids the security guarantees of the deployment and must only be used for testing or
// benchmarking. It should be called before the Server starts serving requests.
func (s *Server) MakeInsecure() {
	s.allowInsecure = true
}

// SetManifest registers a new manifest at the Coordinator.
func (s *Server) SetManifest(ctx context.Context, req *userapi.SetManifestRequest) (*userapi.SetManifestResponse, error) {
	s.logger.Info("SetManifest called")
	resp, _, err := s.setManifest(ctx, req, s.authorizeSignatureOrPeer)
	return resp, err
}

// SetManifestSigned registers a new manifest at the Coordinator, like [Server.SetManifest], for callers
// that can't present a client certificate, like those of the HTTP API.
//
// The request must therefore carry a workload owner signature, verified against the WorkloadOwnerPubKeys
// of the manifest in force for updates, and against those of the requested manifest for the initial manifest.
func (s *Server) SetManifestSigned(ctx context.Context, req *userapi.SetManifestRequest) (*userapi.SetManifestResponse, *stateguard.State, error) {
	s.logger.Info("SetManifestSigned called")
	return s.setManifest(ctx, req, s.authorizeSignature)
}

// authorizer checks that a SetManifest request is authorized by one of the workload owner keys.
type authorizer func(ctx context.Context, keys []manifest.HexString, latestTransitionHash [history.HashSize]byte, initial bool, req *userapi.SetManifestRequest) error

// authorizeSignatureOrPeer authorizes requests of the gRPC API.
func (s *Server) authorizeSignatureOrPeer(ctx context.Context, keys []manifest.HexString, latestTransitionHash [history.HashSize]byte, initial bool, req *userapi.SetManifestRequest) error {
	if initial {
		// The initial manifest is trusted on first use, unless the caller asks for it to be checked.
		if req.Signature == nil {
			return nil
		}
		if err := validateSignature(keys, latestTransitionHash, req); err != nil {
			s.logger.Warn("SetManifest signature validation failed for initial manifest", "err", err)
			return statusErrf(codes.PermissionDenied, ErrInvalidSignature, "validating manifest signature: %v", err)
		}
		return nil
	}

	err := validateSignature(keys, latestTransitionHash, req)
	switch {
	case errors.Is(err, errNoSignature):
		if err := validatePeer(ctx, keys); err != nil {
			s.logger.Warn("SetManifest peer validation failed", "err", err)
			return status.Errorf(codes.PermissionDenied, "validating peer: %v", err)
		}
		return nil
	case err != nil:
		s.logger.Warn("SetManifest signature validation failed", "err", err)
		return statusErrf(codes.PermissionDenied, ErrInvalidSignature, "validating manifest signature: %v", err)
	}
	return nil
}

// authorizeSignature authorizes requests by their signature only.
func (s *Server) authorizeSignature(_ context.Context, keys []manifest.HexString, latestTransitionHash [history.HashSize]byte, initial bool, req *userapi.SetManifestRequest) error {
	if initial && len(keys) == 0 && len(req.Signature) == 0 {
		// There is no key to check against, so the initial manifest is trusted on first use.
		return nil
	}
	if err := validateSignature(keys, latestTransitionHash, req); err != nil {
		s.logger.Warn("SetManifestSigned signature validation failed", "initial", initial, "err", err)
		return statusErrf(codes.PermissionDenied, ErrInvalidSignature, "validating manifest signature: %v", err)
	}
	return nil
}

func (s *Server) setManifest(ctx context.Context, req *userapi.SetManifestRequest, authorize authorizer) (*userapi.SetManifestResponse, *stateguard.State, error) {
	oldState, err := s.guard.GetState(ctx)
	switch {
	case errors.Is(err, stateguard.ErrStaleState):
		return nil, nil, statusErr(codes.FailedPrecondition, ErrNeedsRecovery)
	case errors.Is(err, stateguard.ErrNoState):
		// This is fine, we are going to set the initial manifest.
	case err != nil:
		return nil, nil, status.Errorf(codes.Internal, "getting state: %v", err)
	}

	var m *manifest.Manifest
	if err := json.Unmarshal(req.Manifest, &m); err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "unmarshaling manifest: %v", err)
	}
	if m == nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "manifest is null")
	}

	var resp userapi.SetManifestResponse

	var se *seedengine.SeedEngine
	if oldState != nil {
		oldManifest := oldState.Manifest()
		// Subsequent SetManifest call, check permissions of caller.
		if err := authorize(ctx, oldManifest.WorkloadOwnerPubKeys, oldState.LatestTransition().TransitionHash, false, req); err != nil {
			return nil, nil, err
		}
		se = oldState.SeedEngine()
		if slices.Compare(oldManifest.SeedshareOwnerPubKeys, m.SeedshareOwnerPubKeys) != 0 {
			s.logger.Warn("SetManifest detected attempted seedshare owners change", "from", oldManifest.SeedshareOwnerPubKeys, "to", m.SeedshareOwnerPubKeys)
			return nil, nil, status.Errorf(codes.PermissionDenied, "changes to seedshare owners are not allowed")
		}
		if req.GetPreviousTransitionHash() != nil && !bytes.Equal(oldState.LatestTransition().TransitionHash[:], req.GetPreviousTransitionHash()) {
			return nil, nil, statusErrf(codes.FailedPrecondition, ErrTransitionMismatch, "previous transition hash '%x' does not match latest state '%x'", req.GetPreviousTransitionHash(), oldState.LatestTransition().TransitionHash)
		}
	} else {
		// First SetManifest call, initialize seed engine.
		if err := authorize(ctx, m.WorkloadOwnerPubKeys, [history.HashSize]byte{}, true, req); err != nil {
			return nil, nil, err
		}
		if req.GetPreviousTransitionHash() != nil && !bytes.Equal(req.GetPreviousTransitionHash(), make([]byte, history.HashSize)) {
			return nil, nil, statusErrf(codes.FailedPrecondition, ErrTransitionMismatch, "previous transition hash '%x' requested but manifest history is empty", req.GetPreviousTransitionHash())
		}
		seed, err := cryptohelpers.GenerateRandomBytes(constants.SecretSeedSize)
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, "generating random bytes for seed: %v", err)
		}
		salt, err := cryptohelpers.GenerateRandomBytes(constants.SecretSeedSaltSize)
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, "generating random bytes for seed salt: %v", err)
		}

		seedShares, err := manifest.EncryptSeedShares(seed, m.SeedshareOwnerPubKeys)
		if err != nil {
			return nil, nil, status.Errorf(codes.InvalidArgument, "initializing seed engine: %v", err)
		}
		se, err = seedengine.New(seed, salt)
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, "setting seed: %v", err)
		}
		resp.SeedSharesDoc = &userapi.SeedShareDocument{
			Salt:       salt,
			SeedShares: seedShares,
		}
	}

	if err := s.checkManifestSecurity(m); err != nil {
		s.logger.Warn("SetManifest rejected the manifest", "err", err)
		return nil, nil, statusErr(codes.InvalidArgument, err)
	}

	state, err := s.guard.UpdateState(ctx, oldState, se, req.GetManifest(), req.GetPolicies())
	if err != nil {
		code := codes.Internal
		if errors.Is(err, stateguard.ErrConcurrentUpdate) {
			code = codes.FailedPrecondition
		}
		return nil, nil, statusErrf(code, err, "updating Coordinator state: %v", err)
	}
	resp.MeshCA = state.CA().GetMeshCACert()
	resp.RootCA = state.CA().GetRootCACert()

	s.logger.Info("SetManifest succeeded")
	return &resp, state, nil
}

// GetManifests retrieves the current CA certificates, the manifest history and all policies.
func (s *Server) GetManifests(ctx context.Context, _ *userapi.GetManifestsRequest) (*userapi.GetManifestsResponse, error) {
	s.logger.Info("GetManifest called")
	state, err := s.guard.GetState(ctx)
	switch {
	case errors.Is(err, stateguard.ErrNoState):
		return nil, statusErr(codes.FailedPrecondition, ErrNoManifest)
	case errors.Is(err, stateguard.ErrStaleState):
		return nil, statusErr(codes.FailedPrecondition, ErrNeedsRecovery)
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
		LatestTransition: &userapi.LatestTransition{
			TransitionHash: state.LatestTransition().TransitionHash[:],
			Signature:      state.LatestTransition().Signature,
		},
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
		return nil, statusErr(codes.FailedPrecondition, ErrNoStateToRecover)
	case err != nil:
		return nil, status.Errorf(codes.Internal, "getting state: %v", err)
	default:
		return nil, statusErr(codes.FailedPrecondition, ErrAlreadyRecovered)
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

	_, err = s.guard.ResetState(ctx, oldState, &seedAuthorizer{req: req, checkManifestSecurity: s.checkManifestSecurity})
	if err != nil {
		return nil, fmt.Errorf("resetting state: %w", err)
	}
	return &userapi.RecoverResponse{}, nil
}

// checkManifestSecurity verifies that the manifest doesn't mix secure and insecure platforms and
// that its security level matches the Coordinator's configuration.
func (s *Server) checkManifestSecurity(mnfst *manifest.Manifest) error {
	hasInsecure := mnfst.HasInsecurePlatforms()
	hasSecure := mnfst.HasSecurePlatforms()

	if hasInsecure && hasSecure {
		return ErrMixedManifestNotAllowed
	}
	if hasInsecure && !s.allowInsecure {
		return ErrInsecureNotAllowed
	}
	if !hasInsecure && s.allowInsecure {
		return ErrSecureNotAllowed
	}
	return nil
}

type seedAuthorizer struct {
	req                   *userapi.RecoverRequest
	checkManifestSecurity func(*manifest.Manifest) error
}

func (a *seedAuthorizer) AuthorizeByManifest(ctx context.Context, mnfst *manifest.Manifest, _ [history.HashSize]byte) (*seedengine.SeedEngine, *ecdsa.PrivateKey, error) {
	if err := a.checkManifestSecurity(mnfst); err != nil {
		return nil, nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	if err := validatePeer(ctx, mnfst.SeedshareOwnerPubKeys); err != nil {
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

func validateSignature(keys []manifest.HexString, latestTransitionHash [history.HashSize]byte, req *userapi.SetManifestRequest) error {
	if len(keys) == 0 {
		return errors.New("setting manifest is disabled (no workload owner keys in manifest)")
	}
	if len(req.Signature) == 0 {
		return errNoSignature
	}

	tr := &history.Transition{
		ManifestHash:           history.Digest(req.Manifest),
		PreviousTransitionHash: latestTransitionHash,
	}
	hash := tr.OwnerSigningDigest()

	for _, key := range keys {
		trustedWorkloadOwnerKey, err := manifest.ParseWorkloadOwnerPublicKey(key)
		if err != nil {
			return fmt.Errorf("parsing key: %w", err)
		}
		if ecdsa.VerifyASN1(trustedWorkloadOwnerKey, hash[:], req.Signature) {
			return nil
		}
	}
	return errors.New("invalid manifest signature")
}

func validatePeer(ctx context.Context, keys []manifest.HexString) error {
	if len(keys) == 0 {
		return errors.New("setting manifest is disabled")
	}

	peerPubKey, err := getPeerPublicKey(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		trustedWorkloadOwnerKey, err := key.Bytes()
		if err != nil {
			return fmt.Errorf("parsing key: %w", err)
		}
		if bytes.Equal(peerPubKey, trustedWorkloadOwnerKey) {
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

	// ErrInsecureNotAllowed is returned when a manifest contains insecure platforms but the
	// Coordinator is not configured to allow them.
	ErrInsecureNotAllowed = errors.New("manifest contains insecure platforms, but the coordinator is not configured to allow them")

	// ErrMixedManifestNotAllowed is returned when a manifest mixes secure and insecure platforms.
	// Manifests must be either all-secure or all-insecure.
	ErrMixedManifestNotAllowed = errors.New("manifest must not mix secure and insecure platforms")

	// ErrSecureNotAllowed is returned when a secure manifest is submitted to a Coordinator that is
	// configured to accept only insecure manifests.
	ErrSecureNotAllowed = errors.New("secure manifest is not allowed when the coordinator is configured for insecure manifests")

	// ErrInvalidSignature is returned when the caller is not authorized to change the manifest.
	ErrInvalidSignature = errors.New("manifest signature is invalid")

	// ErrTransitionMismatch is returned when the requested previous transition hash does not match
	// the Coordinator's latest transition, i.e. the compare-and-swap guarding the update failed.
	ErrTransitionMismatch = errors.New("previous transition hash does not match the latest state")

	// ErrNoStateToRecover is returned when recovery is requested but the Coordinator has no state.
	ErrNoStateToRecover = errors.New("no state to recover from")

	errNoSignature = errors.New("manifest signature is empty")
)

func statusErr(code codes.Code, err error) error {
	return &statusError{status: status.New(code, err.Error()), err: err}
}

func statusErrf(code codes.Code, err error, format string, args ...any) error {
	return &statusError{status: status.New(code, fmt.Sprintf(format, args...)), err: err}
}

type statusError struct {
	status *status.Status
	err    error
}

func (e *statusError) Error() string { return e.status.Err().Error() }

func (e *statusError) GRPCStatus() *status.Status { return e.status }

func (e *statusError) Unwrap() error { return e.err }
