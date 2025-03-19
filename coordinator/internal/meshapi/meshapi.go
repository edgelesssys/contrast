// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package meshapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
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

// Recover provides key material to authenticated workloads with the Coordinator role.
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
		return nil, status.Errorf(codes.PermissionDenied, "role %q not allowed to recover", entry.Role)
	}

	ca := state.CA()
	se := state.SeedEngine()

	meshCAPrivKeyPEM, err := encodeKey(ca.GetIntermCAPrivKey())
	if err != nil {
		return nil, fmt.Errorf("encoding mesh CA private key: %w", err)
	}

	resp := &meshapi.RecoverResponse{
		Seed:           se.Seed(),
		Salt:           se.Salt(),
		MeshCAKey:      meshCAPrivKeyPEM,
		LatestManifest: state.ManifestBytes(),
	}

	return resp, nil
}

func encodeKey(key *ecdsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshaling private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	}), nil
}
