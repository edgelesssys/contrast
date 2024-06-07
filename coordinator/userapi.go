// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/edgelesssys/contrast/internal/appendable"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/grpc/atlscredentials"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/memstore"
	"github.com/edgelesssys/contrast/internal/userapi"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type userAPIServer struct {
	grpc            *grpc.Server
	policyTextStore store[manifest.HexString, manifest.Policy]
	manifSetGetter  manifestSetGetter
	logger          *slog.Logger
	mux             sync.RWMutex

	userapi.UnimplementedUserAPIServer
}

func newUserAPIServer(mSGetter manifestSetGetter, reg *prometheus.Registry, log *slog.Logger) *userAPIServer {
	issuer := snp.NewIssuer(logger.NewNamed(log, "snp-issuer"))
	credentials := atlscredentials.New(issuer, nil)

	grpcUserAPIMetrics := grpcprometheus.NewServerMetrics(
		grpcprometheus.WithServerCounterOptions(
			grpcprometheus.WithSubsystem("contrast_userapi"),
		),
		grpcprometheus.WithServerHandlingTimeHistogram(
			grpcprometheus.WithHistogramSubsystem("contrast_userapi"),
			grpcprometheus.WithHistogramBuckets([]float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5}),
		),
	)

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: 15 * time.Second}),
		grpc.ChainStreamInterceptor(
			grpcUserAPIMetrics.StreamServerInterceptor(),
		),
		grpc.ChainUnaryInterceptor(
			grpcUserAPIMetrics.UnaryServerInterceptor(),
		),
	)
	s := &userAPIServer{
		grpc:            grpcServer,
		policyTextStore: memstore.New[manifest.HexString, manifest.Policy](),
		manifSetGetter:  mSGetter,
		logger:          log.WithGroup("userapi"),
	}
	userapi.RegisterUserAPIServer(s.grpc, s)

	grpcUserAPIMetrics.InitializeMetrics(grpcServer)
	reg.MustRegister(grpcUserAPIMetrics)

	return s
}

func (s *userAPIServer) Serve(endpoint string) error {
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	return s.grpc.Serve(lis)
}

func (s *userAPIServer) SetManifest(ctx context.Context, req *userapi.SetManifestRequest,
) (*userapi.SetManifestResponse, error) {
	s.logger.Info("SetManifest called")
	s.mux.Lock()
	defer s.mux.Unlock()

	if err := s.validatePeer(ctx); err != nil {
		s.logger.Warn("SetManifest peer validation failed", "err", err)
		return nil, status.Errorf(codes.PermissionDenied, "validating peer: %v", err)
	}

	var m *manifest.Manifest
	if err := json.Unmarshal(req.Manifest, &m); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unmarshaling manifest: %v", err)
	}

	if len(m.Policies) == 0 {
		return nil, status.Error(codes.InvalidArgument, "manifest must contain at least one policy")
	}
	if len(m.Policies) != len(req.Policies) {
		return nil, status.Error(codes.InvalidArgument, "request must contain exactly the policies referenced in the manifest")
	}

	for _, policyBytes := range req.Policies {
		policy := manifest.Policy(policyBytes)
		if _, ok := m.Policies[policy.Hash()]; !ok {
			return nil, status.Errorf(codes.InvalidArgument, "policy %v not found in manifest", policy.Hash())
		}
		s.policyTextStore.Set(policy.Hash(), policy)
	}

	if err := s.manifSetGetter.SetManifest(m); err != nil {
		return nil, status.Errorf(codes.Internal, "setting manifest: %v", err)
	}

	// TODO(burgerdev): CA should be returned by SetManifest
	_, ca := s.manifSetGetter.GetManifestsAndLatestCA()

	resp := &userapi.SetManifestResponse{
		RootCA: ca.GetRootCACert(),
		MeshCA: ca.GetMeshCACert(),
	}

	s.logger.Info("SetManifest succeeded")
	return resp, nil
}

func (s *userAPIServer) GetManifests(_ context.Context, _ *userapi.GetManifestsRequest,
) (*userapi.GetManifestsResponse, error) {
	s.logger.Info("GetManifest called")
	s.mux.RLock()
	defer s.mux.RUnlock()

	manifests, ca := s.manifSetGetter.GetManifestsAndLatestCA()
	if len(manifests) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "no manifests set")
	}

	manifestBytes, err := manifestSliceToBytesSlice(manifests)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshaling manifests: %v", err)
	}

	policies := s.policyTextStore.GetAll()
	if len(policies) == 0 {
		return nil, status.Error(codes.Internal, "no policies found in store")
	}

	resp := &userapi.GetManifestsResponse{
		Manifests: manifestBytes,
		Policies:  policySliceToBytesSlice(policies),
		RootCA:    ca.GetRootCACert(),
		MeshCA:    ca.GetMeshCACert(),
	}

	s.logger.Info("GetManifest succeeded")
	return resp, nil
}

func (s *userAPIServer) validatePeer(ctx context.Context) error {
	latest, err := s.manifSetGetter.LatestManifest()
	if err != nil && errors.Is(err, appendable.ErrIsEmpty) {
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

func policySliceToBytesSlice(s []manifest.Policy) [][]byte {
	var policies [][]byte
	for _, policy := range s {
		policies = append(policies, policy)
	}
	return policies
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

type manifestSetGetter interface {
	SetManifest(*manifest.Manifest) error
	GetManifestsAndLatestCA() ([]*manifest.Manifest, *ca.CA)
	LatestManifest() (*manifest.Manifest, error)
}

type store[keyT comparable, valueT any] interface {
	Get(key keyT) (valueT, bool)
	GetAll() []valueT
	Set(key keyT, value valueT)
}
