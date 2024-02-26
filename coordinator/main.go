package main

import (
	"fmt"
	"net"
	"os"

	"github.com/edgelesssys/nunki/internal/ca"
	"github.com/edgelesssys/nunki/internal/logger"
	"github.com/edgelesssys/nunki/internal/meshapi"
	"github.com/edgelesssys/nunki/internal/userapi"
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

	caInstance, err := ca.New()
	if err != nil {
		return fmt.Errorf("creating CA: %w", err)
	}

	meshAuth := newMeshAuthority(caInstance, logger)
	userAPI := newUserAPIServer(meshAuth, caInstance, logger)
	meshAPI := newMeshAPIServer(meshAuth, caInstance, logger)

	eg := errgroup.Group{}

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
