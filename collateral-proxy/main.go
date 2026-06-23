// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/edgelesssys/contrast/collateral-proxy/internal/cache"
	"github.com/edgelesssys/contrast/collateral-proxy/internal/proxy"
	"github.com/edgelesssys/contrast/collateral-proxy/internal/upstream"
	"github.com/prometheus/client_golang/prometheus"
)

var version = "0.0.0-dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		addr            = flag.String("addr", ":80", "listen address")
		stateDir        = flag.String("state-dir", "/var/lib/collateral-proxy", "directory for cache state")
		upstreamTimeout = flag.Duration("upstream-timeout", 10*time.Second, "per-request upstream timeout")
	)
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	log.Info("collateral-proxy starting", "version", version, "addr", *addr, "stateDir", *stateDir)

	c, err := cache.New(filepath.Join(*stateDir, "cache"))
	if err != nil {
		return fmt.Errorf("cache init: %w", err)
	}
	fetcher := upstream.New(&http.Client{Timeout: *upstreamTimeout})

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           proxy.New(log, c, fetcher, prometheus.NewRegistry()),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- httpSrv.ListenAndServe() }()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed", "err", err)
		}
	}
	return nil
}
