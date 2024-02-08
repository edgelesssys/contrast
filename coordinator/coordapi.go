package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/edgelesssys/nunki/internal/attestation/snp"
	"github.com/edgelesssys/nunki/internal/coordapi"
	"github.com/edgelesssys/nunki/internal/grpc/atlscredentials"
	"github.com/edgelesssys/nunki/internal/logger"
	"github.com/edgelesssys/nunki/internal/manifest"
	"github.com/edgelesssys/nunki/internal/memstore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type coordAPIServer struct {
	grpc            *grpc.Server
	policyTextStore store[manifest.HexString, manifest.Policy]
	manifSetGetter  manifestSetGetter
	caChainGetter   certChainGetter
	logger          *slog.Logger
	mux             sync.RWMutex

	coordapi.UnimplementedCoordAPIServer
}

func newCoordAPIServer(mSGetter manifestSetGetter, caGetter certChainGetter, log *slog.Logger) *coordAPIServer {
	issuer := snp.NewIssuer(logger.NewNamed(log, "snp-issuer"))
	credentials := atlscredentials.New(issuer, nil)
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: 15 * time.Second}),
	)
	s := &coordAPIServer{
		grpc:            grpcServer,
		policyTextStore: memstore.New[manifest.HexString, manifest.Policy](),
		manifSetGetter:  mSGetter,
		caChainGetter:   caGetter,
		logger:          log.WithGroup("coordapi"),
	}
	coordapi.RegisterCoordAPIServer(s.grpc, s)
	return s
}

func (s *coordAPIServer) Serve(endpoint string) error {
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	return s.grpc.Serve(lis)
}

func (s *coordAPIServer) SetManifest(_ context.Context, req *coordapi.SetManifestRequest,
) (*coordapi.SetManifestResponse, error) {
	s.logger.Info("SetManifest called")
	s.mux.Lock()
	defer s.mux.Unlock()

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

	resp := &coordapi.SetManifestResponse{
		CACert:     s.caChainGetter.GetRootCACert(),
		IntermCert: s.caChainGetter.GetIntermCert(),
	}

	s.logger.Info("SetManifest succeeded")
	return resp, nil
}

func (s *coordAPIServer) GetManifests(_ context.Context, _ *coordapi.GetManifestsRequest,
) (*coordapi.GetManifestsResponse, error) {
	s.logger.Info("GetManifest called")
	s.mux.RLock()
	defer s.mux.RUnlock()

	manifests := s.manifSetGetter.GetManifests()
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

	resp := &coordapi.GetManifestsResponse{
		Manifests:  manifestBytes,
		Policies:   policySliceToBytesSlice(policies),
		CACert:     s.caChainGetter.GetRootCACert(),
		IntermCert: s.caChainGetter.GetIntermCert(),
	}

	s.logger.Info("GetManifest succeeded")
	return resp, nil
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

type certChainGetter interface {
	GetRootCACert() []byte
	GetMeshCACert() []byte
	GetIntermCert() []byte
}

type manifestSetGetter interface {
	SetManifest(*manifest.Manifest) error
	GetManifests() []*manifest.Manifest
}

type store[keyT comparable, valueT any] interface {
	Get(key keyT) (valueT, bool)
	GetAll() []valueT
	Set(key keyT, value valueT)
}
