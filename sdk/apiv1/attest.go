// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package apiv1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/sdk/internal/apiversion"
)

// ErrCoordinatorUninitialized is returned if the Coordinator attests that no manifest has been set yet.
var ErrCoordinatorUninitialized = errors.New("the Coordinator has no manifest")

// ErrCoordinatorNeedsRecovery is returned if the Coordinator attests that it has persisted state but no secrets,
// e.g. after a restart, and needs to be recovered.
var ErrCoordinatorNeedsRecovery = errors.New("the Coordinator needs to be recovered")

// CoordinatorState represents the validated state of the Contrast Coordinator at a fixed point in time.
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

// GetAttestation requests attestation evidence from the Coordinator.
//
// The nonce needs to be exactly 32 bytes, which should come from a CSPRNG.
// Pass the result to [API.ValidateAttestation] before relying on it.
//
// If the SDK client pins a minimum API version newer than v1, no request is sent and the SDK's
// ErrMinimumAPIVersionUnmet is returned.
func (a *API) GetAttestation(ctx context.Context, nonce []byte) ([]byte, error) {
	if len(nonce) != apitypesv1.NonceSize {
		return nil, fmt.Errorf("bad nonce length: got %d, want %d", len(nonce), apitypesv1.NonceSize)
	}
	if err := apiversion.EnforceMinimum(Version, a.minimumAPIVersion()); err != nil {
		return nil, err
	}
	return a.cfg.HTTP.DoJSON(ctx, apitypesv1.AttestMethod, apitypesv1.AttestPath, &apitypesv1.AttestationRequest{Nonce: nonce})
}

// ValidateAttestation validates the Coordinator state returned by [API.GetAttestation].
//
// The input for this function should be the nonce passed into GetAttestation and the byte slice returned by it.
// The attestation may have been fetched by another party.
//
// If this function returns nil, validation passed and the caller can rely on the state.MeshCA
// issuing certificates according to the last entry of state.Manifests.
//
// The attestation carries the digest of the Coordinator's capabilities, which is bound into the report data.
// If the SDK client negotiated the API version, validation compares that digest against the capabilities used
// for the negotiation, and fails with the SDK's ErrAPIVersionDowngrade if they differ.
//
// If the SDK client or the Coordinator's latest manifest pin a minimum API version newer than v1,
// validation fails with the SDK's ErrMinimumAPIVersionUnmet.
//
// If the Coordinator has no state to report, validation fails with [ErrCoordinatorUninitialized] or
// [ErrCoordinatorNeedsRecovery]. Such a Coordinator reports no manifest to take reference values from,
// so the attestation backing these errors isn't validated.
//
// Note: this function does not verify manifest content! It's the callers responsibility to compare
// the latest manifest with an expected manifest, if that exists, or verify that all manifest
// fields match their expectations.
func (a *API) ValidateAttestation(ctx context.Context, nonce []byte, attestation []byte) (*CoordinatorState, error) {
	if len(nonce) != apitypesv1.NonceSize {
		return nil, fmt.Errorf("bad nonce length: got %d, want %d", len(nonce), apitypesv1.NonceSize)
	}

	resp, err := apitypesv1.UnmarshalAttestationResponse(attestation)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling attestation document: %w", err)
	}
	if len(resp.CapabilitiesDigest) == 0 {
		return nil, errors.New("attestation doesn't include a capabilities digest")
	}

	var errStatus error
	switch resp.Status {
	case apitypesv1.StatusActive:
	case apitypesv1.StatusUninitialized:
		errStatus = ErrCoordinatorUninitialized
	case apitypesv1.StatusStale:
		errStatus = ErrCoordinatorNeedsRecovery
	default:
		return nil, fmt.Errorf("attestation reports unknown Coordinator status %q", string(resp.Status))
	}
	if errStatus != nil {
		if len(resp.Manifests) != 0 || len(resp.Policies) != 0 || len(resp.RootCA) != 0 || len(resp.MeshCA) != 0 {
			return nil, fmt.Errorf("attestation reports Coordinator state although its status is %q", string(resp.Status))
		}
		return nil, fmt.Errorf("%w (not validated: there is no manifest to take reference values from)", errStatus)
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
	transitions := history.BuildTransitionChain(resp.Manifests)
	transitionDigest := transitions[len(transitions)-1].Digest()

	validator, err := a.cfg.Validator(&latestManifest)
	if err != nil {
		return nil, fmt.Errorf("getting validators: %w", err)
	}
	reportData, err := apitypesv1.ConstructAttestationReportData(nonce, transitionDigest[:], resp)
	if err != nil {
		return nil, fmt.Errorf("constructing report data: %w", err)
	}
	if err := validator.Validate(ctx, resp.AttestationType, resp.RawAttestationDoc, reportData[:]); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// The capabilities digest is authenticated now, so it tells whether the negotiation can be trusted.
	if a.cfg.NegotiatedCapabilities != nil {
		if negotiated := a.cfg.NegotiatedCapabilities(); negotiated != nil {
			negotiatedDigest := negotiated.Digest()
			if !bytes.Equal(negotiatedDigest[:], resp.CapabilitiesDigest) {
				return nil, fmt.Errorf("%w: the capabilities used for negotiation don't match the capabilities the Coordinator attested to", apiversion.ErrDowngrade)
			}
		}
	}

	for _, minVersion := range []string{a.minimumAPIVersion(), latestManifest.MinimumAPIVersion} {
		if err := apiversion.EnforceMinimum(Version, minVersion); err != nil {
			return nil, err
		}
	}

	return &CoordinatorState{
		Manifests: resp.Manifests,
		Policies:  resp.Policies,
		RootCA:    resp.RootCA,
		MeshCA:    resp.MeshCA,
	}, nil
}
