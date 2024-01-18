package main

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/edgelesssys/nunki/internal/ca"
	"github.com/edgelesssys/nunki/internal/coordapi"
	"github.com/edgelesssys/nunki/internal/intercom"
	"github.com/edgelesssys/nunki/internal/logger"
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

	namespace, ok := os.LookupEnv("NAMESPACE")
	if !ok {
		return errors.New("NAMESPACE environment variable not set")
	}

	caInstance, err := ca.New(namespace)
	if err != nil {
		return fmt.Errorf("creating CA: %w", err)
	}

	meshAuth := newMeshAuthority(caInstance, logger)
	coordS := newCoordAPIServer(meshAuth, caInstance, logger)
	intercomS := newIntercomServer(meshAuth, caInstance, logger)

	eg := errgroup.Group{}

	eg.Go(func() error {
		logger.Info("Coordinator API listening")
		if err := coordS.Serve(net.JoinHostPort("0.0.0.0", coordapi.Port)); err != nil {
			return fmt.Errorf("serving Coordinator API: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		logger.Info("Coordinator intercom listening")
		if err := intercomS.Serve(net.JoinHostPort("0.0.0.0", intercom.Port)); err != nil {
			return fmt.Errorf("serving intercom API: %w", err)
		}
		return nil
	})

	return eg.Wait()
}
