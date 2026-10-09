// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/apitypes"
	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/atls/validators"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/cryptohelpers"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/sdk/apiv1"
)

// GetAttestation requests attestation evidence from the Coordinator's HTTP API.
//
// It uses the attestation endpoint of the newest API version supported by both this SDK and the Coordinator.
// If no version can be negotiated, it uses the unversioned /attest endpoint, unless the expected manifest pins a MinimumAPIVersion.
//
// The nonce needs to be exactly 32 bytes, which should come from a CSPRNG.
func (c *Client) GetAttestation(ctx context.Context, nonce []byte) ([]byte, error) {
	if len(nonce) != cryptohelpers.RNGLengthDefault {
		return nil, fmt.Errorf("bad nonce length: got %d, want %d", len(nonce), cryptohelpers.RNGLengthDefault)
	}

	version, err := c.NegotiateAPIVersion(ctx)
	switch {
	case errors.Is(err, ErrMinimumAPIVersionUnmet):
		return nil, err
	case err != nil:
		if pinErr := enforceMinimumAPIVersion(legacyAPIVersion, c.expectedManifest); pinErr != nil {
			return nil, fmt.Errorf("%w (negotiating API version: %w)", pinErr, err)
		}
		c.log.Debug("Negotiating API version failed, using the unversioned attestation endpoint", "err", err)
		return c.httpapi.DoJSON(ctx, apitypesv1.AttestMethod, apitypesv1.LegacyAttestPath, &apitypesv1.AttestationRequest{Nonce: nonce})
	}

	switch version {
	case apiv1.Version:
		return c.V1().GetAttestation(ctx, nonce)
	default:
		return nil, fmt.Errorf("GetAttestation is not implemented for API version %q", version)
	}
}

// ValidateAttestation validates the Coordinator state returned by [Client.GetAttestation].
//
// The input for this function should be the nonce passed into GetAttestation and the byte slice returned by it.
//
// If this function returns nil, validation passed and the caller can rely on the state.MeshCA
// issuing certificates according to the last entry of state.Manifests.
//
// Attestations from a versioned endpoint carry the digest of the Coordinator's capabilities, which is bound into the report data.
// If the Client negotiated the API version, validation compares that digest against the capabilities used for the negotiation.
// It fails with [ErrAPIVersionDowngrade] if they differ, or if they contain a newer API version this SDK supports than the one that was used.
//
// If the expected manifest or the Coordinator's latest manifest pins a MinimumAPIVersion that is
// newer than the API version the attestation was fetched with, validation fails with
// [ErrMinimumAPIVersionUnmet].
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
		return nil, fmt.Errorf("coordinator state does not include manifests")
	}
	var latestManifest manifest.Manifest
	if err := json.Unmarshal(resp.Manifests[len(resp.Manifests)-1], &latestManifest); err != nil {
		return nil, fmt.Errorf("unmarshalling latest manifest: %w", err)
	}
	if err := latestManifest.Validate(); err != nil {
		return nil, fmt.Errorf("validating latest manifest: %w", err)
	}

	kdsGetter := certcache.NewCachedHTTPSGetter(c.fsstore, certcache.NeverGCTicker, c.log.WithGroup("kds-getter"), c.collateralProxy)
	validatorsFromManifest := func(kdsGetter *certcache.CachedHTTPSGetter, m *manifest.Manifest, log *slog.Logger) (validators.Validator, error) {
		return m.CoordinatorValidator(log, kdsGetter)
	}
	if c.validatorsFromManifestOverride != nil {
		validatorsFromManifest = c.validatorsFromManifestOverride
	}
	validator, err := validatorsFromManifest(kdsGetter, &latestManifest, c.log)
	if err != nil {
		return nil, fmt.Errorf("getting validators: %w", err)
	}

	transitions := history.BuildTransitionChain(resp.Manifests)
	transitionDigest := transitions[len(transitions)-1].Digest()

	attestationVersion, negotiatedCapabilities := c.negotiation()
	var reportData [apitypesv1.ReportDataSize]byte
	switch attestationVersion {
	case legacyAPIVersion:
		reportData = apitypesv1.ConstructReportData(nonce, transitionDigest[:], &resp.CoordinatorState)
	case apiv1.Version:
		if len(resp.CapabilitiesDigest) == 0 {
			return nil, fmt.Errorf("attestation for API version %s doesn't include a capabilities digest", attestationVersion)
		}
		reportData = apitypesv1.ConstructReportDataWithCapabilities(nonce, transitionDigest[:], resp.CapabilitiesDigest, &resp.CoordinatorState)
	default:
		return nil, fmt.Errorf("ValidateAttestation is not implemented for API version %q", attestationVersion)
	}

	if err := validator.Validate(ctx, resp.AttestationType, resp.RawAttestationDoc, reportData[:]); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// The capabilities digest is authenticated now, so it tells whether the negotiation can be trusted.
	if attestationVersion != legacyAPIVersion && negotiatedCapabilities != nil {
		negotiatedDigest := negotiatedCapabilities.Digest()
		if !bytes.Equal(negotiatedDigest[:], resp.CapabilitiesDigest) {
			return nil, fmt.Errorf("%w: the capabilities used for negotiation don't match the capabilities the Coordinator attested to", ErrAPIVersionDowngrade)
		}
		if newestVersion, _ := newestCommonAPIVersion(negotiatedCapabilities); newestVersion != attestationVersion {
			return nil, fmt.Errorf("%w: attestation was fetched with API version %s, but the Coordinator also supports %s", ErrAPIVersionDowngrade, attestationVersion, newestVersion)
		}
	}

	for _, m := range []*manifest.Manifest{c.expectedManifest, &latestManifest} {
		if err := enforceMinimumAPIVersion(attestationVersion, m); err != nil {
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

func (c *Client) negotiation() (string, *apitypes.CapabilitiesResponse) {
	c.negotiateMu.Lock()
	defer c.negotiateMu.Unlock()
	return c.negotiatedVersion, c.negotiatedCapabilities
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
