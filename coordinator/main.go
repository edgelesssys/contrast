// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/coordinator/internal/authority"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/meshapi"
	"github.com/edgelesssys/contrast/internal/recoveryapi"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
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

	hist, err := history.New()
	if err != nil {
		return fmt.Errorf("creating history: %w", err)
	}

	meshAuth := authority.New(hist, promRegistry, logger)
	recoveryAPI := newRecoveryAPIServer(meshAuth, promRegistry, logger)
	userAPI := newUserAPIServer(meshAuth, promRegistry, logger)
	meshAPI := newMeshAPIServer(meshAuth, meshAuth, promRegistry, logger)

	eg := errgroup.Group{}

	recoverable, err := meshAuth.Recoverable()
	if err != nil {
		return fmt.Errorf("checking recoverability: %w", err)
	}
	if recoverable {
		logger.Warn("Coordinator is in recovery mode")

		eg.Go(func() error {
			logger.Info("Coordinator recovery API listening")
			if err := recoveryAPI.Serve(net.JoinHostPort("0.0.0.0", recoveryapi.Port)); err != nil {
				return fmt.Errorf("serving recovery API: %w", err)
			}
			return nil
		})

		recoveryAPI.WaitRecoveryDone()
		logger.Info("Coordinator recovery done")
	}

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
		if err := userAPI.Serve(net.JoinHostPort("0.0.0.0", userapi.Port)); err != nil {
			return fmt.Errorf("serving Coordinator API: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		logger.Info("Coordinator mesh API listening")
		if err := meshAPI.Serve(net.JoinHostPort("0.0.0.0", meshapi.Port)); err != nil {
			return fmt.Errorf("serving mesh API: %w", err)
		}
		return nil
	})

	return eg.Wait()
}
