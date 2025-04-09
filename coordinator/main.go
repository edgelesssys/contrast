// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/coordinator/internal/authority"
	meshapiserver "github.com/edgelesssys/contrast/coordinator/internal/meshapi"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/atls/issuer"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/grpc/atlscredentials"
	loggerpkg "github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/memstore"
	"github.com/edgelesssys/contrast/internal/meshapi"
	"github.com/edgelesssys/contrast/internal/mount"
	"github.com/edgelesssys/contrast/internal/userapi"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"k8s.io/utils/clock"
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
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	logger, err := loggerpkg.Default()
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

	if err := mount.SetupMount(ctx, logger, "/dev/csi0", "/mnt/state"); err != nil {
		return fmt.Errorf("setting up mount: %w", err)
	}

	metricsPort := os.Getenv(metricsPortEnvVar)
	promRegistry := prometheus.NewRegistry()
	serverMetrics := newServerMetrics(promRegistry)

	hist, err := history.New(logger)
	if err != nil {
		return fmt.Errorf("creating history: %w", err)
	}

	meshAuth := authority.New(hist, promRegistry, logger)

	issuer, err := issuer.New(logger)
	if err != nil {
		return fmt.Errorf("creating issuer: %w", err)
	}

	userAPICredentials := atlscredentials.New(issuer, atls.NoValidators, atls.NoMetrics, loggerpkg.NewNamed(logger, "atlscredentials"))
	userAPIServer := newGRPCServer(userAPICredentials, serverMetrics)
	userapi.RegisterUserAPIServer(userAPIServer, meshAuth)
	serverMetrics.InitializeMetrics(userAPIServer)

	month := 30 * 24 * time.Hour
	ticker := clock.RealClock{}.NewTicker(9 * month)
	defer ticker.Stop()
	kdsGetter := certcache.NewCachedHTTPSGetter(memstore.New[string, []byte](), ticker, loggerpkg.NewNamed(logger, "kds-getter-validator"))

	meshAPIcredentials := meshAuth.Credentials(promRegistry, issuer, kdsGetter)
	meshAPIServer := newGRPCServer(meshAPIcredentials, serverMetrics)
	meshapi.RegisterMeshAPIServer(meshAPIServer, meshapiserver.New(logger))
	serverMetrics.InitializeMetrics(meshAPIServer)

	metricsServer := &http.Server{}

	eg, ctx := errgroup.WithContext(ctx)

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
		metricsServer.Addr = ":" + metricsPort
		metricsServer.Handler = mux
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Serving Prometheus /metrics endpoint", "err", err)
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
		if err := userAPIServer.Serve(lis); err != nil {
			logger.Error("Serving Coordinator API", "err", err)
			return fmt.Errorf("serving Coordinator API: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		logger.Info("Coordinator mesh API listening")
		lis, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", meshapi.Port))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
		if err := meshAPIServer.Serve(lis); err != nil {
			logger.Error("Serving Coordinator API", "err", err)
			return fmt.Errorf("serving Coordinator API: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		logger.Info("Watching manifest store")
		if err := meshAuth.WatchHistory(ctx); err != nil {
			logger.Error("Watching manifest store", "err", err)
		}
		return err
	})

	eg.Go(func() error {
		<-ctx.Done()
		logger.Info("Context done, shutting down", "err", ctx.Err())
		// New context for cleanup, Kubernetes grace period is 30 seconds.
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		wg := &sync.WaitGroup{}
		gracefulStopGRPC(ctx, wg, userAPIServer) //nolint:contextcheck
		gracefulStopGRPC(ctx, wg, meshAPIServer) //nolint:contextcheck
		wg.Wait()
		return metricsServer.Shutdown(ctx) //nolint:contextcheck
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

func newGRPCServer(credentials credentials.TransportCredentials, serverMetrics *grpcprometheus.ServerMetrics) *grpc.Server {
	return grpc.NewServer(
		grpc.Creds(credentials),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: 15 * time.Second}),
		grpc.ChainStreamInterceptor(
			serverMetrics.StreamServerInterceptor(),
		),
		grpc.ChainUnaryInterceptor(
			serverMetrics.UnaryServerInterceptor(),
		),
	)
}

func gracefulStopGRPC(ctx context.Context, wg *sync.WaitGroup, server *grpc.Server) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		cleanupDone := make(chan struct{})
		go func() {
			server.GracefulStop()
			close(cleanupDone)
		}()
		select {
		case <-ctx.Done():
			server.Stop()
		case <-cleanupDone:
		}
	}()
}
