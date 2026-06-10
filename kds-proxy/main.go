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

	"github.com/edgelesssys/contrast/kds-proxy/internal/ca"
	"github.com/edgelesssys/contrast/kds-proxy/internal/cache"
	"github.com/edgelesssys/contrast/kds-proxy/internal/proxy"
	"github.com/edgelesssys/contrast/kds-proxy/internal/upstream"
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
		addr            = flag.String("addr", ":3128", "listen address")
		stateDir        = flag.String("state-dir", "/var/lib/kds-proxy", "directory for CA and cache state")
		upstreamTimeout = flag.Duration("upstream-timeout", 10*time.Second, "per-request upstream timeout")
	)
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	log.Info("kds-proxy starting", "version", version, "addr", *addr, "stateDir", *stateDir)

	authority, err := ca.LoadOrGenerate(filepath.Join(*stateDir, "ca"))
	if err != nil {
		return fmt.Errorf("CA init: %w", err)
	}
	c, err := cache.New(filepath.Join(*stateDir, "cache"))
	if err != nil {
		return fmt.Errorf("cache init: %w", err)
	}
	fetcher := upstream.New(&http.Client{Timeout: *upstreamTimeout})

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           proxy.New(log, authority, c, fetcher, nil),
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed", "err", err)
		}
	}
	return nil
}
