package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/edgelesssys/nunki/internal/attestation/snp"
	"github.com/edgelesssys/nunki/internal/coordapi"
	"github.com/edgelesssys/nunki/internal/grpc/atlscredentials"
	"github.com/edgelesssys/nunki/internal/manifest"
	"github.com/edgelesssys/nunki/internal/memstore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type coordAPIServer struct {
	grpc            *grpc.Server
	policyTextStore store[manifest.HexString, manifest.Policy]
	manifSetGetter  manifestSetGetter
	caChainGetter   certChainGetter
	logger          *slog.Logger

	coordapi.UnimplementedCoordAPIServer
}

func newCoordAPIServer(mSGetter manifestSetGetter, caGetter certChainGetter, log *slog.Logger) (*coordAPIServer, error) {
	issuer := snp.NewIssuer(log)
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
	return s, nil
}

func (i *coordAPIServer) Serve(endpoint string) error {
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return i.grpc.Serve(lis)
}

func (s *coordAPIServer) SetManifest(ctx context.Context, req *coordapi.SetManifestRequest,
) (*coordapi.SetManifestResponse, error) {
	s.logger.Info("SetManifest called")

	var m *manifest.Manifest
	if err := json.Unmarshal(req.Manifest, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %v", err)
	}

	for _, policyBytes := range req.Policies {
		policy := manifest.Policy(policyBytes)
		if _, ok := m.Policies[policy.Hash()]; !ok {
			return nil, fmt.Errorf("policy %v not found in manifest", policy.Hash())
		}
		s.policyTextStore.Set(policy.Hash(), policy)
	}

	if err := s.manifSetGetter.SetManifest(m); err != nil {
		return nil, err
	}

	resp := &coordapi.SetManifestResponse{
		CACert:     s.caChainGetter.GetCACert(),
		IntermCert: s.caChainGetter.GetIntermCert(),
	}

	s.logger.Info("SetManifest succeeded")
	return resp, nil
}

func (s *coordAPIServer) GetManifests(ctx context.Context, _ *coordapi.GetManifestsRequest,
) (*coordapi.GetManifestsResponse, error) {
	s.logger.Info("GetManifest called")

	manifests := s.manifSetGetter.GetManifests()
	if len(manifests) == 0 {
		return nil, fmt.Errorf("no manifests found")
	}

	manifestBytes, err := manifestSliceToBytesSlice(manifests)
	if err != nil {
		return nil, err
	}

	resp := &coordapi.GetManifestsResponse{
		Manifests:  manifestBytes,
		Policies:   policySliceToBytesSlice(s.policyTextStore.GetAll()),
		CACert:     s.caChainGetter.GetCACert(),
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
			return nil, fmt.Errorf("mashaling manifest %d manifest: %v", i, err)
		}
		manifests = append(manifests, manifestBytes)
	}
	return manifests, nil
}

type certChainGetter interface {
	GetCACert() []byte
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
