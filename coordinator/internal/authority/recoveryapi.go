// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/grpc/atlscredentials"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/recoveryapi"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type recoveryAPIServer struct {
	grpc          *grpc.Server
	logger        *slog.Logger
	recoverable   recoverable
	recoveryDoneC chan struct{}

	recoveryapi.UnimplementedRecoveryAPIServer
}

func newRecoveryAPIServer(recoveryTarget recoverable, reg *prometheus.Registry, log *slog.Logger) *recoveryAPIServer {
	issuer := snp.NewIssuer(logger.NewNamed(log, "snp-issuer"))
	credentials := atlscredentials.New(issuer, nil)

	grpcUserAPIMetrics := grpcprometheus.NewServerMetrics(
		grpcprometheus.WithServerCounterOptions(
			grpcprometheus.WithSubsystem("contrast_recoveryapi"),
		),
		grpcprometheus.WithServerHandlingTimeHistogram(
			grpcprometheus.WithHistogramSubsystem("contrast_recoveryapi"),
			grpcprometheus.WithHistogramBuckets([]float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5}),
		),
	)

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: 15 * time.Second}),
		grpc.ChainStreamInterceptor(
			grpcUserAPIMetrics.StreamServerInterceptor(),
		),
		grpc.ChainUnaryInterceptor(
			grpcUserAPIMetrics.UnaryServerInterceptor(),
		),
	)
	s := &recoveryAPIServer{
		grpc:          grpcServer,
		logger:        log.WithGroup("recoveryapi"),
		recoverable:   recoveryTarget,
		recoveryDoneC: make(chan struct{}),
	}
	recoveryapi.RegisterRecoveryAPIServer(s.grpc, s)

	grpcUserAPIMetrics.InitializeMetrics(grpcServer)
	reg.MustRegister(grpcUserAPIMetrics)

	return s
}

func (s *recoveryAPIServer) Serve(endpoint string) error {
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", endpoint, err)
	}
	return s.grpc.Serve(lis)
}

func (s *recoveryAPIServer) WaitRecoveryDone() {
	<-s.recoveryDoneC
}

func (s *recoveryAPIServer) Recover(_ context.Context, req *recoveryapi.RecoverRequest) (*recoveryapi.RecoverResponse, error) {
	return &recoveryapi.RecoverResponse{}, s.recoverable.Recover(req.Seed, req.Salt)
}

type recoverable interface {
	Recover(seed, salt []byte) error
}
