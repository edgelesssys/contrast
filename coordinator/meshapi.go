// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"k8s.io/utils/clock"
)

type meshAPIServer struct {
	grpc          *grpc.Server
	certGet       certGetter
	caChainGetter certChainGetter
	ticker        clock.Ticker
	logger        *slog.Logger

	meshapi.UnimplementedMeshAPIServer
}

type certGetter interface {
	GetCert(peerPublicKeyHashStr string) ([]byte, error)
}

func newMeshAPIServer(meshAuth *meshAuthority, caGetter certChainGetter, reg *prometheus.Registry, log *slog.Logger) *meshAPIServer {
	ticker := clock.RealClock{}.NewTicker(24 * time.Hour)
	kdsGetter := snp.NewCachedHTTPSGetter(memstore.New[string, []byte](), ticker, logger.NewNamed(log, "kds-getter"))

	attestationFailuresCounter := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Subsystem: "contrast_meshapi",
		Name:      "attestation_failures",
		Help:      "Number of attestation failures from workloads to the Coordinator.",
	})

	validator := snp.NewValidatorWithCallbacks(meshAuth, kdsGetter, logger.NewNamed(log, "snp-validator"), attestationFailuresCounter, meshAuth)
	credentials := atlscredentials.New(atls.NoIssuer, []atls.Validator{validator})

	grpcMeshAPIMetrics := grpcprometheus.NewServerMetrics(
		grpcprometheus.WithServerCounterOptions(
			grpcprometheus.WithSubsystem("contrast_meshapi"),
		),
		grpcprometheus.WithServerHandlingTimeHistogram(
			grpcprometheus.WithHistogramSubsystem("contrast_meshapi"),
			grpcprometheus.WithHistogramBuckets([]float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5}),
		),
	)

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: 15 * time.Second}),
		grpc.ChainStreamInterceptor(
			grpcMeshAPIMetrics.StreamServerInterceptor(),
		),
		grpc.ChainUnaryInterceptor(
			grpcMeshAPIMetrics.UnaryServerInterceptor(),
		),
	)
	s := &meshAPIServer{
		grpc:          grpcServer,
		certGet:       meshAuth,
		caChainGetter: caGetter,
		ticker:        ticker,
		logger:        log.WithGroup("meshapi"),
	}
	meshapi.RegisterMeshAPIServer(s.grpc, s)

	grpcMeshAPIMetrics.InitializeMetrics(grpcServer)
	reg.MustRegister(grpcMeshAPIMetrics)

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

func (i *meshAPIServer) NewMeshCert(_ context.Context, req *meshapi.NewMeshCertRequest,
) (*meshapi.NewMeshCertResponse, error) {
	i.logger.Info("NewMeshCert called")

	cert, err := i.certGet.GetCert(req.PeerPublicKeyHash)
	if err != nil {
		return nil, status.Errorf(codes.Internal,
			"getting certificate with public key hash %q: %v", req.PeerPublicKeyHash, err)
	}

	meshCACert := i.caChainGetter.GetMeshCACert()
	intermCert := i.caChainGetter.GetIntermCACert()

	return &meshapi.NewMeshCertResponse{
		MeshCACert: meshCACert,
		CertChain:  append(cert, intermCert...),
		RootCACert: i.caChainGetter.GetRootCACert(),
	}, nil
}
