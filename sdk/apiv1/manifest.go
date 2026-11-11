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

// SetManifestResult is the validated outcome of [API.SetManifest].
type SetManifestResult struct {
	// CoordinatorState is the state the Coordinator is in after setting the manifest.
	// Its LatestTransitionSignature is empty, as the Coordinator doesn't report it when setting a manifest.
	CoordinatorState
	// SeedSharesDoc is the JSON-encoded seed share document, in the format `contrast recover` reads.
	// It's only set if the request set the initial manifest, and is required to recover the Coordinator.
	SeedSharesDoc []byte
}

// SetManifest sets a manifest at the Coordinator.
//
// The nonce of the request must be 32 random bytes, which should come from a CSPRNG.
// The Coordinator attests its response under that nonce, and SetManifest validates it against the reference values
// of the manifest being set. If it returns nil, the caller can rely on the returned MeshCA issuing certificates
// according to the manifest. Errors that occur after the Coordinator accepted the request don't undo the update.
//
// If the manifest or the SDK client pin a minimum API version newer than v1, no request is sent and the
// SDK's ErrMinimumAPIVersionUnmet is returned. This keeps the caller from setting a manifest that it can't
// talk to the Coordinator about afterward.
func (a *API) SetManifest(ctx context.Context, req *apitypesv1.SetManifestRequest) (*SetManifestResult, error) {
	if req == nil {
		return nil, errors.New("request must not be nil")
	}
	if len(req.Nonce) != apitypesv1.NonceSize {
		return nil, fmt.Errorf("bad nonce length: got %d, want %d", len(req.Nonce), apitypesv1.NonceSize)
	}
	var m manifest.Manifest
	if err := json.Unmarshal(req.Manifest, &m); err != nil {
		return nil, fmt.Errorf("unmarshalling manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}
	if err := apiversion.EnforceMinimum(Version, a.cfg.MinimumAPIVersion); err != nil {
		return nil, err
	}
	if err := apiversion.EnforceMinimum(Version, m.MinimumAPIVersion); err != nil {
		return nil, fmt.Errorf("refusing to set manifest: %w", err)
	}

	body, err := a.cfg.HTTP.DoJSON(ctx, apitypesv1.SetManifestMethod, apitypesv1.ManifestPath, req)
	if err != nil {
		return nil, fmt.Errorf("setting manifest: %w", err)
	}
	var resp apitypesv1.SetManifestResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}
	return a.validateSetManifestResponse(ctx, req, &m, &resp)
}

// validateSetManifestResponse validates that resp attests the manifest m of req being set at a genuine Coordinator.
func (a *API) validateSetManifestResponse(ctx context.Context, req *apitypesv1.SetManifestRequest, m *manifest.Manifest, resp *apitypesv1.SetManifestResponse) (*SetManifestResult, error) {
	if len(resp.Manifests) == 0 {
		return nil, errors.New("coordinator state does not include manifests")
	}
	if !bytes.Equal(req.Manifest, resp.Manifests[len(resp.Manifests)-1]) {
		return nil, errors.New("latest manifest reported by the Coordinator isn't the one that was set")
	}
	transitions := history.BuildTransitionChain(resp.Manifests)
	latest := transitions[len(transitions)-1]
	if len(req.PreviousTransitionHash) != 0 && !bytes.Equal(req.PreviousTransitionHash, latest.PreviousTransitionHash[:]) {
		return nil, fmt.Errorf("manifest was set after transition %x, not after the requested %x", latest.PreviousTransitionHash, req.PreviousTransitionHash)
	}
	transitionDigest := latest.Digest()

	// The manifest being set is the expected one, so its reference values are the ones to validate against.
	validator, err := a.cfg.Validator(m)
	if err != nil {
		return nil, fmt.Errorf("getting validators: %w", err)
	}
	reportData, err := apitypesv1.ConstructSetManifestReportData(req.Nonce, transitionDigest[:], resp)
	if err != nil {
		return nil, fmt.Errorf("constructing report data: %w", err)
	}
	if err := validator.Validate(ctx, resp.AttestationType, resp.RawAttestationDoc, reportData[:]); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &SetManifestResult{
		CoordinatorState: CoordinatorState{
			Manifests:            resp.Manifests,
			Policies:             resp.Policies,
			RootCA:               resp.RootCA,
			MeshCA:               resp.MeshCA,
			LatestTransitionHash: transitionDigest[:],
		},
		SeedSharesDoc: resp.SeedSharesDoc,
	}, nil
}
