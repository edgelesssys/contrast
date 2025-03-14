// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/edgelesssys/contrast/coordinator/internal/authority"
	"github.com/edgelesssys/contrast/internal/atls/issuer"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/meshapi"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/edgelesssys/contrast/internal/attestation/extension"
	"github.com/edgelesssys/contrast/internal/oid"
)

type meshAPIServer struct {
	grpc    *grpc.Server
	cleanup func()
	logger  *slog.Logger

	meshapi.UnimplementedMeshAPIServer
}

func newMeshAPIServer(meshAuth *authority.Authority, reg *prometheus.Registry, serverMetrics *grpcprometheus.ServerMetrics, log *slog.Logger,
) (*meshAPIServer, error) {
	issuer, err := issuer.New(log)
	if err != nil {
		return nil, fmt.Errorf("creating issuer: %w", err)
	}
	credentials, cancel := meshAuth.Credentials(reg, issuer)

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: 15 * time.Second}),
		grpc.ChainStreamInterceptor(
			serverMetrics.StreamServerInterceptor(),
		),
		grpc.ChainUnaryInterceptor(
			serverMetrics.UnaryServerInterceptor(),
		),
	)
	s := &meshAPIServer{
		grpc:    grpcServer,
		cleanup: cancel,
		logger:  log.WithGroup("meshapi"),
	}
	meshapi.RegisterMeshAPIServer(s.grpc, s)
	serverMetrics.InitializeMetrics(s.grpc)

	return s, nil
}

func (i *meshAPIServer) Serve(endpoint string) error {
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	defer i.cleanup()
	return i.grpc.Serve(lis)
}

// NewMeshCert creates a mesh certificate for the connected peer.
//
// When this handler is called, the transport credentials already ensured that
// the peer is authorized according to the manifest, so it can start issuing
// right away.
func (i *meshAPIServer) NewMeshCert(ctx context.Context, _ *meshapi.NewMeshCertRequest) (*meshapi.NewMeshCertResponse, error) {
	i.logger.Info("NewMeshCert called")

	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get peer from context")
	}

	authInfo, ok := p.AuthInfo.(authority.AuthInfo)
	if !ok {
		return nil, fmt.Errorf("unexpected AuthInfo type: %T", p.AuthInfo)
	}
	state := authInfo.State
	report := authInfo.Report
	tlsInfo := authInfo.TLSInfo

	if len(tlsInfo.State.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no peer certificates found")
	}

	peerCert := tlsInfo.State.PeerCertificates[0]
	peerPubKeyBytes, err := x509.MarshalPKIXPublicKey(peerCert.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not marshal public key: %w", err)
	}

	hostData := manifest.NewHexString(report.HostData())
	entry, ok := state.Manifest.Policies[hostData]
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "policy hash %s not found in manifest", hostData)
	}
	dnsNames := entry.SANs

	peerPubKey, err := x509.ParsePKIXPublicKey(peerPubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse peer public key: %w", err)
	}

	extensions, err := report.ClaimsToCertExtension()
	if err != nil {
		return nil, fmt.Errorf("failed to construct extensions: %w", err)
	}
	if entry.WorkloadSecretID != "" {
		workloadSecretExtension, err := extension.ConvertExtension(extension.NewBytesExtension(oid.WorkloadSecretOID,
			[]byte(entry.WorkloadSecretID)))
		if err != nil {
			return nil, fmt.Errorf("failed to construct workload secret extension: %w", err)
		}
		extensions = append(extensions, workloadSecretExtension)
	}
	cert, err := state.CA.NewAttestedMeshCert(dnsNames, extensions, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to issue new attested mesh cert: %w", err)
	}

	resp := &meshapi.NewMeshCertResponse{
		MeshCACert: state.CA.GetMeshCACert(),
		CertChain:  append(cert, state.CA.GetIntermCACert()...),
		RootCACert: state.CA.GetRootCACert(),
	}

	if entry.WorkloadSecretID != "" {
		workloadSecret, err := state.SeedEngine.DeriveWorkloadSecret(entry.WorkloadSecretID)
		if err != nil {
			return nil, fmt.Errorf("failed to derive workload secret: %w", err)
		}
		resp.WorkloadSecret = workloadSecret
	}

	return resp, nil
}
