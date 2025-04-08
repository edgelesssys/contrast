// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/edgelesssys/contrast/coordinator/history"
	meshapiserver "github.com/edgelesssys/contrast/coordinator/internal/meshapi"
	"github.com/edgelesssys/contrast/coordinator/internal/probes"
	transitengine "github.com/edgelesssys/contrast/coordinator/internal/transitengineapi"
	userapiserver "github.com/edgelesssys/contrast/coordinator/internal/userapi"
	"github.com/edgelesssys/contrast/coordinator/stateguard"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/atls/issuer"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/defaultdeny"
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
	metricsEnvVar       = "CONTRAST_METRICS"
	probeAndMetricsPort = 9102
	// transitEngineAPIPort specifies the default port to expose the transit engine API.
	transitEngineAPIPort = "8200"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (retErr error) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	fmt.Fprintf(os.Stderr, "Contrast Coordinator %s\n", constants.Version)
	fmt.Fprintln(os.Stderr, "Report issues at https://github.com/edgelesssys/contrast/issues")

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

	// The coordinator doesn't have an initcontainer to remove the default deny rule.
	// Since we control the code of the coordinator, we never need the service-mesh sidecar
	// and therefore can just remove the default deny rule.
	logger.Info("removing default deny rule")
	if err := defaultdeny.RemoveDefaultDenyRule(logger); err != nil {
		return fmt.Errorf("removing default deny rule: %w", err)
	}

	if err := mount.SetupMount(ctx, logger, "/dev/csi0", "/mnt/state"); err != nil {
		return fmt.Errorf("setting up mount: %w", err)
	}

	promRegistry := prometheus.NewRegistry()
	serverMetrics := newServerMetrics(promRegistry)

	hist, err := history.New(logger)
	if err != nil {
		return fmt.Errorf("creating history: %w", err)
	}

	meshAuth := stateguard.New(hist, promRegistry, logger)

	issuer, err := issuer.New(logger)
	if err != nil {
		return fmt.Errorf("creating issuer: %w", err)
	}

	userAPICredentials := atlscredentials.New(issuer, atls.NoValidators, atls.NoMetrics, loggerpkg.NewNamed(logger, "atlscredentials"))
	userAPIServer := newGRPCServer(userAPICredentials, serverMetrics)
	userapi.RegisterUserAPIServer(userAPIServer, userapiserver.New(logger, meshAuth))
	serverMetrics.InitializeMetrics(userAPIServer)

	month := 30 * 24 * time.Hour
	ticker := clock.RealClock{}.NewTicker(9 * month)
	defer ticker.Stop()
	kdsGetter := certcache.NewCachedHTTPSGetter(memstore.New[string, []byte](), ticker, loggerpkg.NewNamed(logger, "kds-getter-validator"))

	meshAPIcredentials := meshAuth.Credentials(promRegistry, issuer, kdsGetter)
	meshAPIServer := newGRPCServer(meshAPIcredentials, serverMetrics)
	meshapi.RegisterMeshAPIServer(meshAPIServer, meshapiserver.New(logger))
	serverMetrics.InitializeMetrics(meshAPIServer)

	httpServer := &http.Server{}

	startupHandler := probes.StartupHandler{UserapiStarted: false, MeshapiStarted: false}
	livenessHandler := probes.LivenessHandler{Hist: hist}
	readinessHandler := probes.ReadinessHandler{Guard: meshAuth}

	userapiStarted := &startupHandler.UserapiStarted
	meshapiStarted := &startupHandler.MeshapiStarted

	transitAPIServer, err := transitengine.NewTransitEngineAPI(meshAuth, logger)

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		_, enableMetrics := os.LookupEnv(metricsEnvVar)
		mux := http.NewServeMux()
		if enableMetrics {
			logger.Info("Starting prometheus /metrics endpoint on port " + strconv.Itoa(probeAndMetricsPort))
			mux.Handle("/metrics", promhttp.InstrumentMetricHandler(
				promRegistry, promhttp.HandlerFor(
					promRegistry,
					promhttp.HandlerOpts{Registry: promRegistry},
				),
			))
		}
		mux.Handle("/probe/startup", &startupHandler)
		mux.Handle("/probe/liveness", &livenessHandler)
		mux.Handle("/probe/readiness", &readinessHandler)
		httpServer.Addr = ":" + strconv.Itoa(probeAndMetricsPort)
		httpServer.Handler = mux
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Starting http server", "err", err)
			return fmt.Errorf("starting http server: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		logger.Info("Coordinator user API listening")
		lis, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", userapi.Port))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
		*userapiStarted = true
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
		*meshapiStarted = true
		if err := meshAPIServer.Serve(lis); err != nil {
			logger.Error("Serving Coordinator API", "err", err)
			return fmt.Errorf("serving Coordinator API: %w", err)
		}
		return nil
	})

	registerEnterpriseServices(ctx, eg, &components{
		guard:       meshAuth,
		logger:      logger,
		httpsGetter: kdsGetter,
		issuer:      issuer,
	})

	eg.Go(func() error {
		logger.Info("Coordinator transit engine API listening")
		lis, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", transitEngineAPIPort))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
		if err := transitAPIServer.ServeTLS(lis, "", ""); err != nil {
			logger.Error("Serving transit engine API", "err", err)
			return fmt.Errorf("serving transit engine API: %w", err)
		}
		return nil
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
		_ = transitAPIServer.Shutdown(ctx) //nolint:contextcheck
		return httpServer.Shutdown(ctx)    //nolint:contextcheck
	})

	return eg.Wait()
}

type components struct {
	logger      *slog.Logger
	guard       *stateguard.Guard
	issuer      atls.Issuer
	httpsGetter *certcache.CachedHTTPSGetter
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
