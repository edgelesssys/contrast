// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/meshapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// NewMeshCert creates a mesh certificate for the connected peer.
//
// When this handler is called, the transport credentials already ensured that
// the peer is authorized according to the manifest, so it can start issuing
// right away.
func (a *Authority) NewMeshCert(ctx context.Context, _ *meshapi.NewMeshCertRequest) (*meshapi.NewMeshCertResponse, error) {
	a.logger.Info("NewMeshCert called")

	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get peer from context")
	}

	authInfo, ok := p.AuthInfo.(AuthInfo)
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
