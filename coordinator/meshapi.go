// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/edgelesssys/contrast/coordinator/internal/authority"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/grpc/atlscredentials"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/memstore"
	"github.com/edgelesssys/contrast/internal/meshapi"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/peer"
	"k8s.io/utils/clock"
)

type meshAPIServer struct {
	grpc         *grpc.Server
	bundleGetter certBundleGetter
	ticker       clock.Ticker
	logger       *slog.Logger

	meshapi.UnimplementedMeshAPIServer
}

type certBundleGetter interface {
	GetCertBundle(peerPublicKeyHashStr string) (authority.Bundle, error)
}

func newMeshAPIServer(meshAuth *authority.Authority, bundleGetter certBundleGetter, reg *prometheus.Registry, serverMetrics *grpcprometheus.ServerMetrics, log *slog.Logger) *meshAPIServer {
	ticker := clock.RealClock{}.NewTicker(24 * time.Hour)
	kdsGetter := snp.NewCachedHTTPSGetter(memstore.New[string, []byte](), ticker, logger.NewNamed(log, "kds-getter"))

	attestationFailuresCounter := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Subsystem: "contrast_meshapi",
		Name:      "attestation_failures_total",
		Help:      "Number of attestation failures from workloads to the Coordinator.",
	})

	validator := snp.NewValidatorWithCallbacks(meshAuth, kdsGetter,
		logger.NewWithAttrs(logger.NewNamed(log, "validator"), map[string]string{"tee-type": "snp"}),
		attestationFailuresCounter, meshAuth)
	credentials := atlscredentials.New(atls.NoIssuer, []atls.Validator{validator})

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
		grpc:         grpcServer,
		bundleGetter: bundleGetter,
		ticker:       ticker,
		logger:       log.WithGroup("meshapi"),
	}
	meshapi.RegisterMeshAPIServer(s.grpc, s)
	serverMetrics.InitializeMetrics(s.grpc)

	return s
}

func (i *meshAPIServer) Serve(endpoint string) error {
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	defer i.ticker.Stop()
	return i.grpc.Serve(lis)
}

func (i *meshAPIServer) NewMeshCert(ctx context.Context, _ *meshapi.NewMeshCertRequest) (*meshapi.NewMeshCertResponse, error) {
	i.logger.Info("NewMeshCert called")

	// Fetch the peer public key from gRPC's TLS context and look up the corresponding cetificate.

	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get peer from context")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, fmt.Errorf("failed to get TLS info from peer")
	}

	if len(tlsInfo.State.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no peer certificates found")
	}

	peerCert := tlsInfo.State.PeerCertificates[0]
	peerPubKeyBytes, err := x509.MarshalPKIXPublicKey(peerCert.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not marshal public key: %w", err)
	}
	peerPubKeyHash := sha256.Sum256(peerPubKeyBytes)
	peerPublicKeyHashStr := hex.EncodeToString(peerPubKeyHash[:])

	bundle, err := i.bundleGetter.GetCertBundle(peerPublicKeyHashStr)
	if err != nil {
		return nil, fmt.Errorf("server did not create a bundle for ")
	}

	return &meshapi.NewMeshCertResponse{
		MeshCACert: bundle.MeshCA,
		CertChain:  append(bundle.WorkloadCert, bundle.IntermediateCA...),
		RootCACert: bundle.RootCA,
	}, nil
}
