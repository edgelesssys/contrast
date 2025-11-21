// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	meshapiserver "github.com/edgelesssys/contrast/coordinator/internal/meshapi"
	"github.com/edgelesssys/contrast/coordinator/internal/peerdiscovery"
	"github.com/edgelesssys/contrast/coordinator/internal/peerrecovery"
	"github.com/edgelesssys/contrast/coordinator/internal/probes"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	transitengine "github.com/edgelesssys/contrast/coordinator/internal/transitengineapi"
	userapiserver "github.com/edgelesssys/contrast/coordinator/internal/userapi"
	"github.com/edgelesssys/contrast/coordinator/internal/verify"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/atls/issuer"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/defaultdeny"
	"github.com/edgelesssys/contrast/internal/grpc/atlscredentials"
	"github.com/edgelesssys/contrast/internal/history"
	loggerpkg "github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/memstore"
	"github.com/edgelesssys/contrast/internal/meshapi"
	"github.com/edgelesssys/contrast/internal/userapi"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/clock"
)

const (
	metricsEnvVar       = "CONTRAST_METRICS"
	verifyPort          = 1314
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
	ctxSignal, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
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

	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("getting kubeconfig: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("creating Kubernetes clientset: %w", err)
	}
	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return fmt.Errorf("reading namespace file: %w", err)
	}
	discovery := peerdiscovery.New(clientset, string(namespace))

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
	userapi.RegisterUserAPIServer(userAPIServer, userapiserver.New(logger, meshAuth, discovery))
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

	var userapiStarted, meshapiStarted, recoveryStarted atomic.Bool

	startupHandler := probes.StartupHandler{
		UserapiStarted:  &userapiStarted,
		MeshapiStarted:  &meshapiStarted,
		RecoveryStarted: &recoveryStarted,
	}
	readinessHandler := probes.ReadinessHandler{Guard: meshAuth}

	transitAPIServer, err := transitengine.NewTransitEngineAPI(meshAuth, logger)
	if err != nil {
		return fmt.Errorf("creating transit engine API server: %w", err)
	}

	eg, ctx := errgroup.WithContext(ctxSignal)

	eg.Go(func() error {
		h := verify.Handler{
			Issuer:     issuer,
			StateGuard: meshAuth,
		}

		httpServer := &http.Server{}
		mux := http.NewServeMux()
		mux.Handle("/verify", &h)

		httpServer.Addr = ":" + strconv.Itoa(verifyPort)
		httpServer.Handler = mux
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Starting verify http server", "err", err)
			return fmt.Errorf("starting verify http server: %w", err)
		}
		return nil
	})

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
		mux.Handle("/probe/liveness", &startupHandler)
		mux.Handle("/probe/readiness", &readinessHandler)
		httpServer.Addr = ":" + strconv.Itoa(probeAndMetricsPort)
		httpServer.Handler = mux
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Starting probes and metrics http server", "err", err)
			return fmt.Errorf("starting probes and metrics http server: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		logger.Info("Coordinator user API listening")
		lis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", net.JoinHostPort("0.0.0.0", userapi.Port))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
		userapiStarted.Store(true)
		if err := userAPIServer.Serve(lis); err != nil {
			logger.Error("Serving Coordinator API", "err", err)
			return fmt.Errorf("serving Coordinator API: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		logger.Info("Coordinator mesh API listening")
		lis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", net.JoinHostPort("0.0.0.0", meshapi.Port))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
		meshapiStarted.Store(true)
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
		return nil
	})

	eg.Go(func() error {
		logger.Info("Coordinator peer recovery started")
		recoverer := peerrecovery.New(meshAuth, discovery, issuer, kdsGetter, logger)

		onceCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		if err := recoverer.RecoverOnce(onceCtx); err != nil {
			logger.Warn("Running initial peer recovery", "err", err)
		}
		cancel()
		recoveryStarted.Store(true)

		if err := recoverer.RunRecovery(ctx); err != nil {
			logger.Error("Running peer recovery", "err", err)
			return fmt.Errorf("running peer recovery: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		logger.Info("Coordinator transit engine API listening")
		lis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", net.JoinHostPort("0.0.0.0", transitEngineAPIPort))
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
		if ctxSignal.Err() != nil {
			logger.Info("Received signal, shutting down")
		} else {
			logger.Info("Context done, shutting down", "err", ctx.Err())
		}
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
