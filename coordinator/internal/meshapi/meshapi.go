// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package meshapi

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/coordinator/internal/authority"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/meshapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Server implements the meshapi service.
type Server struct {
	logger *slog.Logger

	meshapi.UnimplementedMeshAPIServer
}

// New returns a meshapi server using a sub-logger of log.
func New(log *slog.Logger) *Server {
	return &Server{
		logger: log.WithGroup("meshapi"),
	}
}

// NewMeshCert creates a mesh certificate for the connected peer.
//
// When this handler is called, the transport credentials already ensured that
// the peer is authorized according to the manifest, so it can start issuing
// right away.
func (i *Server) NewMeshCert(ctx context.Context, _ *meshapi.NewMeshCertRequest) (*meshapi.NewMeshCertResponse, error) {
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
	entry, ok := state.Manifest().Policies[hostData]
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
	ca := state.CA()
	cert, err := ca.NewAttestedMeshCert(dnsNames, extensions, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to issue new attested mesh cert: %w", err)
	}

	resp := &meshapi.NewMeshCertResponse{
		MeshCACert: ca.GetMeshCACert(),
		CertChain:  append(cert, ca.GetIntermCACert()...),
		RootCACert: ca.GetRootCACert(),
	}

	if entry.WorkloadSecretID != "" {
		workloadSecret, err := state.SeedEngine().DeriveWorkloadSecret(entry.WorkloadSecretID)
		if err != nil {
			return nil, fmt.Errorf("failed to derive workload secret: %w", err)
		}
		resp.WorkloadSecret = workloadSecret
	}

	return resp, nil
}

// Recover verifies the peers policy and role and recovers the calling Coordinator.
func (i *Server) Recover(ctx context.Context, _ *meshapi.RecoverRequest) (*meshapi.RecoverResponse, error) {
	i.logger.Info("Recover called")

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

	hostData := manifest.NewHexString(report.HostData())
	entry, ok := state.Manifest().Policies[hostData]
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "policy hash %s not found in manifest", hostData)
	}
	if entry.Role != manifest.RoleCoordinator {
		return nil, status.Errorf(codes.PermissionDenied, "role %s not allowed to recover", entry.Role)
	}

	ca := state.CA()
	se := state.SeedEngine()

	rootCAPrivKeyDER, err := x509.MarshalECPrivateKey(se.RootCAKey())
	if err != nil {
		return nil, fmt.Errorf("marshaling root CA private key: %w", err)
	}
	rootCAPrivKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: rootCAPrivKeyDER,
	})

	meshCAPrivKeyDER, err := x509.MarshalECPrivateKey(ca.GetIntermCAPrivKey())
	if err != nil {
		return nil, fmt.Errorf("marshaling mesh CA private key: %w", err)
	}
	meshCAPrivKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: meshCAPrivKeyDER,
	})

	mnfst, err := json.Marshal(state.Manifest())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	resp := &meshapi.RecoverResponse{
		Seed:           se.Seed(),
		Salt:           se.Salt(),
		RootCAKey:      rootCAPrivKeyPEM,
		RootCACert:     ca.GetRootCACert(),
		MeshCAKey:      meshCAPrivKeyPEM,
		MeshCACert:     ca.GetMeshCACert(),
		LatestManifest: mnfst,
	}

	return resp, nil
}
