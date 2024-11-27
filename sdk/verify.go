// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
)

// Client is used to interact with a Contrast deployment.
type Client struct {
	log *slog.Logger
}

// New returns a Client with a default logger.
// Configures logging via slog to write to stderr.
func New() Client {
	return Client{
		log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
	}
}

// NewWithSlog can be used to configure how the SDK logs messages.
func NewWithSlog(log *slog.Logger) Client {
	return Client{
		log: log,
	}
}

// GetCoordinatorState calls GetManifests on the coordinator's userapi via aTLS.
func (c Client) GetCoordinatorState(ctx context.Context, kdsDir string, manifestBytes []byte, endpoint string, policyHash []byte) (CoordinatorState, error) {
	var m manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return CoordinatorState{}, fmt.Errorf("unmarshalling manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return CoordinatorState{}, fmt.Errorf("validating manifest: %w", err)
	}

	validators, err := ValidatorsFromManifest(kdsDir, &m, c.log, policyHash)
	if err != nil {
		return CoordinatorState{}, fmt.Errorf("getting validators: %w", err)
	}
	dialer := dialer.New(atls.NoIssuer, validators, atls.NoMetrics, &net.Dialer{})

	c.log.Debug("Dialing coordinator", "endpoint", endpoint)

	conn, err := dialer.Dial(ctx, endpoint)
	if err != nil {
		return CoordinatorState{}, fmt.Errorf("dialing coordinator: %w", err)
	}
	defer conn.Close()

	c.log.Debug("Getting manifest")

	client := userapi.NewUserAPIClient(conn)
	resp, err := client.GetManifests(ctx, &userapi.GetManifestsRequest{})
	if err != nil {
		return CoordinatorState{}, fmt.Errorf("getting manifests: %w", err)
	}

	return CoordinatorState{
		Manifests: resp.Manifests,
		Policies:  resp.Policies,
		RootCA:    resp.RootCA,
		MeshCA:    resp.MeshCA,
	}, nil
}

// Verify checks if a given manifest is the latest manifest in the given history.
// The expected manifest should be supplied by the verifying party, the history should be received from the coordinator.
func (Client) Verify(expectedManifest []byte, manifestHistory [][]byte) error {
	if len(manifestHistory) == 0 {
		return fmt.Errorf("manifest history is empty")
	}

	currentManifest := manifestHistory[len(manifestHistory)-1]
	if !bytes.Equal(currentManifest, expectedManifest) {
		return fmt.Errorf("active manifest does not match expected manifest")
	}

	return nil
}

// CoordinatorState contains the Coordinator's response to a GetManifests call.
type CoordinatorState struct {
	// Manifests is a slice of manifests. It represents the manifest history of the Coordinator it was received from.
	Manifests [][]byte
	// Policies is a slice of policies. It contains all policies that have been referenced in any of the manifests in the manifest history. Used to verify the guarantees a deployment had over its lifetime.
	Policies [][]byte
	// PEM-encoded certificate of the deployment's root CA.
	RootCA []byte
	// PEM-encoded certificate of the deployment's mesh CA.
	MeshCA []byte
}
