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

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
)

// Client is used to interact with a Contrast deployment.
type Client struct {
	log *slog.Logger
}

// New returns a Client.
func New(log *slog.Logger) Client {
	return Client{
		log: log,
	}
}

// Verify checks if a given manifest is the latest manifest in the given history.
func Verify(expected []byte, history [][]byte) error {
	currentManifest := history[len(history)-1]
	if !bytes.Equal(currentManifest, expected) {
		return fmt.Errorf("active manifest does not match expected manifest")
	}

	return nil
}

// GetManifests calls GetManifests on the coordinator's userapi.
func (c Client) GetManifests(ctx context.Context, manifestBytes []byte, endpoint string, policyHash []byte) (GetManifestsResponse, error) {
	var m manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return GetManifestsResponse{}, fmt.Errorf("unmarshalling manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return GetManifestsResponse{}, fmt.Errorf("validating manifest: %w", err)
	}

	validators, err := ValidatorsFromManifest(&m, c.log, policyHash)
	if err != nil {
		return GetManifestsResponse{}, fmt.Errorf("getting validators: %w", err)
	}
	dialer := dialer.New(atls.NoIssuer, validators, atls.NoMetrics, &net.Dialer{})

	c.log.Debug("Dialing coordinator", "endpoint", endpoint)

	conn, err := dialer.Dial(ctx, endpoint)
	if err != nil {
		return GetManifestsResponse{}, fmt.Errorf("dialing coordinator: %w", err)
	}
	defer conn.Close()

	c.log.Debug("Getting manifest")

	client := userapi.NewUserAPIClient(conn)
	resp, err := client.GetManifests(ctx, &userapi.GetManifestsRequest{})
	if err != nil {
		return GetManifestsResponse{}, fmt.Errorf("getting manifests: %w", err)
	}

	return GetManifestsResponse{
		Manifests: resp.Manifests,
		Policies:  resp.Policies,
		RootCA:    resp.RootCA,
		MeshCA:    resp.MeshCA,
	}, nil
}

// GetManifestsResponse contains the Coordinator's response to a GetManifests call.
type GetManifestsResponse struct {
	Manifests [][]byte
	Policies  [][]byte
	// PEM-encoded certificate
	RootCA []byte
	// PEM-encoded certificate
	MeshCA []byte
}
