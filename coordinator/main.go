// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/meshapi"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
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

	caInstance, err := ca.New()
	if err != nil {
		return fmt.Errorf("creating CA: %w", err)
	}

	promRegistry := prometheus.NewRegistry()

	meshAuth := newMeshAuthority(caInstance, logger)
	userAPI := newUserAPIServer(meshAuth, caInstance, promRegistry, logger)
	meshAPI := newMeshAPIServer(meshAuth, caInstance, promRegistry, logger)

	eg := errgroup.Group{}

	eg.Go(func() error {
		logger.Info("Starting prometheus /metrics endpoint")
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.InstrumentMetricHandler(
			promRegistry, promhttp.HandlerFor(
				promRegistry,
				promhttp.HandlerOpts{Registry: promRegistry},
			),
		))
		if err := http.ListenAndServe(":9102", mux); err != nil {
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
