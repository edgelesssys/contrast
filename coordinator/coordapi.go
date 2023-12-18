package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/katexochen/coordinator-kbs/internal/attestation/snp"
	"github.com/katexochen/coordinator-kbs/internal/coordapi"
	"github.com/katexochen/coordinator-kbs/internal/grpc/atlscredentials"
	"github.com/katexochen/coordinator-kbs/internal/manifest"
	"github.com/katexochen/coordinator-kbs/internal/memstore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type coordAPIServer struct {
	grpc            *grpc.Server
	policyTextStore store[manifest.HexString, manifest.Policy]
	mSetter         manifestSetter
	caChainGetter   certChainGetter

	coordapi.UnimplementedCoordAPIServer
}

func newCoordAPIServer(mSetter manifestSetter, caGetter certChainGetter) (*coordAPIServer, error) {
	issuer := snp.NewIssuer()
	credentials := atlscredentials.New(issuer, nil)
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: 15 * time.Second}),
	)
	s := &coordAPIServer{
		grpc:            grpcServer,
		policyTextStore: memstore.New[manifest.HexString, manifest.Policy](),
		mSetter:         mSetter,
		caChainGetter:   caGetter,
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
	log.Println("SetManifest called")

	manifestDec, err := base64.StdEncoding.DecodeString(req.Manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %v", err)
	}
	var m *manifest.Manifest
	if err := json.Unmarshal(manifestDec, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %v", err)
	}

	for _, policyBytes := range req.Policies {
		policy := manifest.Policy(policyBytes)
		if _, ok := m.Policies[policy.Hash()]; !ok {
			return nil, fmt.Errorf("policy %v not found in manifest", policy.Hash())
		}
		s.policyTextStore.Set(policy.Hash(), policy)
	}

	if err := s.mSetter.SetManifest(m); err != nil {
		return nil, err
	}

	log.Println("SetManifest succeeded")
	return &coordapi.SetManifestResponse{CACert: s.caChainGetter.GetCACert(), IntermCert: s.caChainGetter.GetIntermCert()}, nil
}

type certChainGetter interface {
	GetCACert() []byte
	GetIntermCert() []byte
}

type manifestSetter interface {
	SetManifest(*manifest.Manifest) error
}

type store[keyT comparable, valueT any] interface {
	Get(key keyT) (valueT, bool)
	Set(key keyT, value valueT)
}

type manifestSetGetter struct {
	setOnce   sync.Once
	manifestC chan *manifest.Manifest
}

func newManifestSetGetter() *manifestSetGetter {
	return &manifestSetGetter{manifestC: make(chan *manifest.Manifest, 1)}
}

func (m *manifestSetGetter) SetManifest(ma *manifest.Manifest) error {
	m.setOnce.Do(func() {
		m.manifestC <- ma
		close(m.manifestC)
	})
	return nil
}

func (m *manifestSetGetter) GetManifest() *manifest.Manifest {
	return <-m.manifestC
}
