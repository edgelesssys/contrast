// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/containerd/ttrpc"
	"github.com/containers/storage/pkg/reexec"
	"github.com/edgelesssys/contrast/imagepuller/internal/api"
	"github.com/edgelesssys/contrast/imagepuller/internal/service"
)

var (
	version = "0.0.0-dev"
	log     *slog.Logger
)

func main() {
	if reexec.Init() {
		return
	}
	flag.Parse()

	log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	if err := os.MkdirAll(filepath.Dir(api.Socket), os.ModePerm); err != nil {
		return logErrAndReturn("Failed to create directory for socket", err)
	}
	if err := os.Remove(api.Socket); err != nil && !errors.Is(err, os.ErrNotExist) {
		return logErrAndReturn("Failed to remove existing socket", err)
	}
	l, err := net.Listen("unix", api.Socket)
	if err != nil {
		return logErrAndReturn("Failed to listen on socket", err)
	}
	defer l.Close()
	defer os.RemoveAll(api.Socket)

	s, err := ttrpc.NewServer()
	if err != nil {
		return logErrAndReturn("Failed to create ttRPC server", err)
	}
	defer s.Close()

	sigintCh := make(chan os.Signal, 1)
	signal.Notify(sigintCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	api.RegisterImagePullServiceService(s, &service.ImagePullerService{
		Error: func(msg string, err error) error {
			errCh <- fmt.Errorf("%s: %w", msg, err)
			return err
		},
		Logger: log,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Info("Started image-puller", "version", version, "socket", api.Socket)
		log.Info("Waiting for image pull request...")
		if err := s.Serve(ctx, l); err != nil {
			log.Error("Failed to start the ttRPC server", "error", err)
		}
	}()

	go func() {
		for err := range errCh {
			if err != nil {
				log.Error("An error occurred while serving the request", "error", err)
			}
		}
	}()

	sig := <-sigintCh
	log.Info("Received signal, shutting down.", "signal", sig)
	close(sigintCh)
	close(errCh)

	if err := s.Shutdown(context.Background()); err != nil {
		return logErrAndReturn("Failed to shut down the ttRPC server", err)
	}
	return nil
}

func logErrAndReturn(msg string, err error) error {
	if err != nil {
		log.Error(msg, "error", err)
		return err
	}
	return nil
}

func toJSON(a any) string {
	bs, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		log.Error("Failed to marshal json", "error", err)
	}
	return string(bs)
}
