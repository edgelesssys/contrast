package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/edgelesssys/nunki/internal/ca"
	"github.com/edgelesssys/nunki/internal/coordapi"
	"github.com/edgelesssys/nunki/internal/intercom"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (retErr error) {
	logger := slog.Default()
	defer func() {
		if retErr != nil {
			logger.Error(retErr.Error())
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

	manifestSetGetter := newManifestSetGetter()

	coordS, err := newCoordAPIServer(manifestSetGetter, caInstance, logger)
	if err != nil {
		return fmt.Errorf("creating coordinator API server: %w", err)
	}

	go func() {
		logger.Info("Coordinator API listening")
		if err := coordS.Serve(net.JoinHostPort("0.0.0.0", coordapi.Port)); err != nil {
			// TODO: collect error using errgroup.
			logger.Error("Coordinator API failed to serve: %v", err)
		}
	}()

	logger.Info("Waiting for manifest")
	manifest := manifestSetGetter.GetManifest()
	logger.Info("Got manifest")

	meshAuth, err := newMeshAuthority(caInstance, manifest, logger)
	if err != nil {
		return fmt.Errorf("creating mesh authority: %v", err)
	}

	intercomS, err := newIntercomServer(meshAuth, caInstance, logger)
	if err != nil {
		return fmt.Errorf("creating intercom server: %v", err)
	}

	logger.Info("Coordinator intercom listening")
	if err := intercomS.Serve(net.JoinHostPort("0.0.0.0", intercom.Port)); err != nil {
		return fmt.Errorf("serving intercom: %v", err)
	}
	return nil
}
