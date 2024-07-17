// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/coordinator/internal/authority"
	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/grpc/atlscredentials"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/meshapi"
	"github.com/edgelesssys/contrast/internal/userapi"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	metricsPortEnvVar = "CONTRAST_METRICS_PORT"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (retErr error) {
	logger, err := logger.Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: creating logger: %v\n", err)
		return err
	}
	defer func() {
		if retErr != nil {
			logger.Error("Coordinator terminated after failure", "err", retErr)
		}
	}()

	logger.Info("Coordinator started")

	if err := setupMount(context.Background(), logger); err != nil {
		return fmt.Errorf("setting up mount: %w", err)
	}

	metricsPort := os.Getenv(metricsPortEnvVar)
	promRegistry := prometheus.NewRegistry()
	serverMetrics := newServerMetrics(promRegistry)

	hist, err := history.New()
	if err != nil {
		return fmt.Errorf("creating history: %w", err)
	}

	meshAuth := authority.New(hist, promRegistry, logger)
	grpcServer, err := newGRPCServer(serverMetrics, logger)
	if err != nil {
		return fmt.Errorf("creating gRPC server: %w", err)
	}

	userapi.RegisterUserAPIServer(grpcServer, meshAuth)
	serverMetrics.InitializeMetrics(grpcServer)

	eg := errgroup.Group{}

	eg.Go(func() error {
		if metricsPort == "" {
			return nil
		}
		if metricsPort == userapi.Port || metricsPort == meshapi.Port {
			return fmt.Errorf("invalid port for metrics endpoint: %s", metricsPort)
		}
		logger.Info("Starting prometheus /metrics endpoint on port " + metricsPort)
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.InstrumentMetricHandler(
			promRegistry, promhttp.HandlerFor(
				promRegistry,
				promhttp.HandlerOpts{Registry: promRegistry},
			),
		))
		if err := http.ListenAndServe(":"+metricsPort, mux); err != nil {
			return fmt.Errorf("serving Prometheus endpoint: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		logger.Info("Coordinator user API listening")
		lis, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", userapi.Port))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
		if err := grpcServer.Serve(lis); err != nil {
			return fmt.Errorf("serving Coordinator API: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		meshAPI := newMeshAPIServer(meshAuth, meshAuth, promRegistry, serverMetrics, logger)
		logger.Info("Coordinator mesh API listening")
		if err := meshAPI.Serve(net.JoinHostPort("0.0.0.0", meshapi.Port)); err != nil {
			return fmt.Errorf("serving mesh API: %w", err)
		}
		return nil
	})

	return eg.Wait()
}

func newServerMetrics(reg *prometheus.Registry) *grpcprometheus.ServerMetrics {
	serverMetrics := grpcprometheus.NewServerMetrics(
		grpcprometheus.WithServerCounterOptions(
			grpcprometheus.WithSubsystem("contrast"),
		),
		grpcprometheus.WithServerHandlingTimeHistogram(
			grpcprometheus.WithHistogramSubsystem("contrast"),
			grpcprometheus.WithHistogramBuckets([]float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5}),
		),
	)
	reg.MustRegister(serverMetrics)
	return serverMetrics
}

func newGRPCServer(serverMetrics *grpcprometheus.ServerMetrics, log *slog.Logger) (*grpc.Server, error) {
	issuer, err := attestation.PlatformIssuer(log)
	if err != nil {
		return nil, fmt.Errorf("creating issuer: %w", err)
	}

	credentials := atlscredentials.New(issuer, nil)

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
	return grpcServer, nil
}
