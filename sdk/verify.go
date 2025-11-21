// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
)

// Client is used to interact with a Contrast deployment.
type Client struct {
	log *slog.Logger
}

// New returns a Client with logging disabled.
func New() Client {
	return Client{
		log: slog.New(slog.DiscardHandler),
	}
}

// NewWithSlog can be used to configure how the SDK logs messages.
func NewWithSlog(log *slog.Logger) Client {
	return Client{
		log: log,
	}
}

// GetCoordinatorState calls GetManifests on the coordinator's userapi via aTLS.
func (c Client) GetCoordinatorState(ctx context.Context, kdsDir string, manifestBytes []byte, endpoint string) (CoordinatorState, error) {
	var m manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return CoordinatorState{}, fmt.Errorf("unmarshalling manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return CoordinatorState{}, fmt.Errorf("validating manifest: %w", err)
	}

	kdsCache := fsstore.New(kdsDir, c.log.WithGroup("kds-cache"))
	kdsGetter := certcache.NewCachedHTTPSGetter(kdsCache, certcache.NeverGCTicker, c.log.WithGroup("kds-getter"))
	validators, err := ValidatorsFromManifest(kdsGetter, &m, c.log)
	if err != nil {
		return CoordinatorState{}, fmt.Errorf("getting validators: %w", err)
	}
	dialer := dialer.New(atls.NoIssuer, validators, atls.NoMetrics, nil, c.log)

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
// The expected manifest should be supplied by the caller, the history should be received from the coordinator.
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

// CoordinatorState represents the state of the Contrast Coordinator at a fixed point in time.
type CoordinatorState struct {
	// Manifests is a slice of manifests. It represents the manifest history of the Coordinator it was received from, sorted from oldest to newest.
	Manifests [][]byte `json:"manifests"`
	// Policies contains all policies that have been referenced in any manifest in Manifests. Used to verify the guarantees a deployment had over its lifetime.
	Policies [][]byte `json:"policies"`
	// PEM-encoded certificate of the deployment's root CA.
	RootCA []byte `json:"root_ca"`
	// PEM-encoded certificate of the deployment's mesh CA.
	MeshCA []byte `json:"mesh_ca"`
}

// ConstructReportData constructs an extended report data digest,
// intended for use with application-level verification.
func ConstructReportData(nonce []byte, transitionDigest []byte, state *CoordinatorState) [64]byte {
	// reportdata = sha256(nonce || sha256(transition) || sha256(root-ca) || sha256(mesh-ca))
	rootCADigest := history.Digest(state.RootCA)
	meshCADigest := history.Digest(state.MeshCA)

	reportdata := append([]byte{}, nonce...)
	reportdata = append(reportdata, transitionDigest[:]...)
	reportdata = append(reportdata, rootCADigest[:]...)
	reportdata = append(reportdata, meshCADigest[:]...)
	hash32 := history.Digest(reportdata)

	var hash64 [64]byte
	copy(hash64[:], hash32[:])

	return hash64
}
