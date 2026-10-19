// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/atls/validators"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/cryptohelpers"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
)

// GetAttestation requests attestation evidence from the unversioned /attest endpoint of the Coordinator's HTTP API.
//
// The versioned API is used through the client of the respective version, like [Client.V1].
// If a minimum API version is pinned, see [Client.WithMinimumAPIVersion], no request is sent and
// [ErrMinimumAPIVersionUnmet] is returned.
//
// The nonce needs to be exactly 32 bytes, which should come from a CSPRNG.
func (c *Client) GetAttestation(ctx context.Context, nonce []byte) ([]byte, error) {
	if len(nonce) != cryptohelpers.RNGLengthDefault {
		return nil, fmt.Errorf("bad nonce length: got %d, want %d", len(nonce), cryptohelpers.RNGLengthDefault)
	}
	if err := enforceMinimumAPIVersion(legacyAPIVersion, c.minimumAPIVersion); err != nil {
		return nil, err
	}
	return c.httpapi.DoJSON(ctx, apitypesv1.AttestMethod, apitypesv1.LegacyAttestPath, &apitypesv1.AttestationRequest{Nonce: nonce})
}

// ValidateAttestation validates the Coordinator state returned by [Client.GetAttestation].
//
// The input for this function should be the nonce passed into GetAttestation and the byte slice returned by it.
//
// If this function returns nil, validation passed and the caller can rely on the state.MeshCA
// issuing certificates according to the last entry of state.Manifests.
//
// If [Client.WithMinimumAPIVersion] or the Coordinator's latest manifest pins a minimum API version,
// validation fails with [ErrMinimumAPIVersionUnmet], as the unversioned endpoint doesn't meet any pin.
//
// Note: this function does not verify manifest content! It's the callers responsibility to compare
// the latest manifest with an expected manifest, if that exists, or verify that all manifest
// fields match their expectations.
func (c *Client) ValidateAttestation(ctx context.Context, nonce []byte, attestation []byte) (*CoordinatorState, error) {
	if len(nonce) != cryptohelpers.RNGLengthDefault {
		return nil, fmt.Errorf("wrong nonce length: got %d, want %d", len(nonce), cryptohelpers.RNGLengthDefault)
	}

	resp, err := apitypesv1.UnmarshalAttestationResponse(attestation)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling attestation document: %w", err)
	}

	if len(resp.Manifests) == 0 {
		return nil, errors.New("coordinator state does not include manifests")
	}
	var latestManifest manifest.Manifest
	if err := json.Unmarshal(resp.Manifests[len(resp.Manifests)-1], &latestManifest); err != nil {
		return nil, fmt.Errorf("unmarshalling latest manifest: %w", err)
	}
	if err := latestManifest.Validate(); err != nil {
		return nil, fmt.Errorf("validating latest manifest: %w", err)
	}

	validator, err := c.coordinatorValidator(&latestManifest)
	if err != nil {
		return nil, fmt.Errorf("getting validators: %w", err)
	}

	transitions := history.BuildTransitionChain(resp.Manifests)
	transitionDigest := transitions[len(transitions)-1].Digest()
	reportData := apitypesv1.ConstructReportData(nonce, transitionDigest[:], &resp.CoordinatorState)
	if err := validator.Validate(ctx, resp.AttestationType, resp.RawAttestationDoc, reportData[:]); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	for _, minVersion := range []string{c.minimumAPIVersion, latestManifest.MinimumAPIVersion} {
		if err := enforceMinimumAPIVersion(legacyAPIVersion, minVersion); err != nil {
			return nil, err
		}
	}

	state := CoordinatorState{
		Manifests: resp.Manifests,
		Policies:  resp.Policies,
		RootCA:    resp.RootCA,
		MeshCA:    resp.MeshCA,
	}
	return &state, nil
}

// CoordinatorState represents the state of the Contrast Coordinator at a fixed point in time.
type CoordinatorState struct {
	// Manifests is a slice of manifests. It represents the manifest history of the Coordinator it was received from, sorted from oldest to newest.
	Manifests [][]byte
	// Policies contains all policies that have been referenced in any manifest in Manifests. Used to verify the guarantees a deployment had over its lifetime.
	Policies [][]byte
	// PEM-encoded certificate of the deployment's root CA.
	RootCA []byte
	// PEM-encoded certificate of the deployment's mesh CA.
	MeshCA []byte
	// Hash of the latest transition in the Coordinator's history.
	LatestTransitionHash []byte
	// Signature of the latest transition hash by the Coordinator.
	LatestTransitionSignature []byte
}

// coordinatorValidator returns the validator for a Coordinator running manifest m.
func (c *Client) coordinatorValidator(m *manifest.Manifest) (validators.Validator, error) {
	kdsGetter := certcache.NewCachedHTTPSGetter(c.fsstore, certcache.NeverGCTicker, c.log.WithGroup("kds-getter"), c.collateralProxy)
	if c.validatorsFromManifestOverride != nil {
		return c.validatorsFromManifestOverride(kdsGetter, m, c.log)
	}
	return m.CoordinatorValidator(c.log, kdsGetter)
}
