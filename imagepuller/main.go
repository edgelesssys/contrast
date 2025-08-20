// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/containerd/ttrpc"
	"github.com/containers/storage/pkg/reexec"
	"github.com/edgelesssys/contrast/imagepuller/internal/api"
	"github.com/edgelesssys/contrast/imagepuller/internal/remote"
	"github.com/edgelesssys/contrast/imagepuller/internal/service"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var version = "0.0.0-dev"

func main() {
	// Reexec handles subprocesses by re-running this binary with argv[0] set to a registered name.
	// If argv[0] matches, Init runs the corresponding handler and returns true to exit immediately.
	// Used by the storage pkg.
	if reexec.Init() {
		return
	}

	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "imagepuller",
		Short:        "pull and mount images",
		Version:      version,
		SilenceUsage: true,
		RunE:         run,
	}
	cmd.Flags().String("storepath", "", "temporary directory to use for storage")
	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	ctxSignal, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	fmt.Fprintf(os.Stderr, "Contrast image-puller %s\n", version)
	fmt.Fprintln(os.Stderr, "Report issues at https://github.com/edgelesssys/contrast/issues")

	if err := os.MkdirAll(filepath.Dir(api.Socket), os.ModePerm); err != nil {
		return fmt.Errorf("creating directory for socket: %w", err)
	}
	if err := os.Remove(api.Socket); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing existing socket: %w", err)
	}

	l, err := (&net.ListenConfig{}).Listen(ctxSignal, "unix", api.Socket)
	if err != nil {
		return fmt.Errorf("listening on socket: %w", err)
	}
	defer l.Close()
	defer os.RemoveAll(api.Socket)

	s, err := ttrpc.NewServer()
	if err != nil {
		return fmt.Errorf("creating ttRPC server: %w", err)
	}
	defer s.Close()

	api.RegisterImagePullServiceService(s, &service.ImagePullerService{
		Logger:            log,
		StorePathOverride: cmd.Flag("storepath").Value.String(),
		Remote:            remote.DefaultRemote{},
	})

	eg, ctx := errgroup.WithContext(ctxSignal)

	eg.Go(func() error {
		log.Info("Started image-puller", "socket", api.Socket)
		log.Info("Waiting for image pull request...")
		if err := s.Serve(ctx, l); err != nil {
			return fmt.Errorf("starting the ttRPC server: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		<-ctx.Done()
		if ctxSignal.Err() != nil {
			log.Info("Received signal, shutting down.")
		} else {
			log.Info("Context done, shutting down", "err", ctx.Err())
		}
		ctxCleanup, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if err := s.Shutdown(ctxCleanup); err != nil { //nolint:contextcheck
			return fmt.Errorf("shutting down the ttRPC server: %w", err)
		}
		return nil
	})

	return eg.Wait()
}
