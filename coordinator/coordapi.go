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
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type coordAPIServer struct {
	grpc          *grpc.Server
	mSetter       manifestSetter
	caChainGetter certChainGetter

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
		grpc:          grpcServer,
		mSetter:       mSetter,
		caChainGetter: caGetter,
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

	if err := s.mSetter.SetManifest(req.Manifest); err != nil {
		return nil, err
	}

	log.Println("SetManifest succeeded")
	return &coordapi.SetManifestResponse{CertChain: s.caChainGetter.GetCertChain()}, nil
}

type certChainGetter interface {
	GetCertChain() [][]byte
}

type manifestSetter interface {
	SetManifest(string) error
}

type manifestSetGetter struct {
	setOnce   sync.Once
	manifestC chan *manifest.Manifest
}

func newManifestSetGetter() *manifestSetGetter {
	return &manifestSetGetter{manifestC: make(chan *manifest.Manifest, 1)}
}

func (m *manifestSetGetter) SetManifest(manifestStr string) error {
	manifestDec, err := base64.StdEncoding.DecodeString(manifestStr)
	if err != nil {
		return fmt.Errorf("failed to decode manifest: %v", err)
	}
	var manifest *manifest.Manifest
	if err := json.Unmarshal(manifestDec, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %v", err)
	}
	m.setOnce.Do(func() {
		m.manifestC <- manifest
		close(m.manifestC)
	})
	return nil
}

func (m *manifestSetGetter) GetManifest() *manifest.Manifest {
	return <-m.manifestC
}
