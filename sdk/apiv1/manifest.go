// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
)

// ManifestPath is the path of the manifest endpoint, relative to the API's base URL.
const ManifestPath = "/v1/manifest"

// SetManifest sets a manifest at the Coordinator and returns the deployment's CA certificates.
//
// The returned SeedSharesDoc is only set if this call set the initial manifest. It holds the
// secret seed, encrypted for each seedshare owner, and is required to recover the deployment.
//
// Note: this function does not verify that the Coordinator is trustworthy! Callers should
// verify it via the SDK's GetAttestation and ValidateAttestation before relying on the response.
func (a *API) SetManifest(ctx context.Context, req *apitypesv1.SetManifestRequest) (*apitypesv1.SetManifestResponse, error) {
	if req == nil {
		return nil, errors.New("request must not be nil")
	}

	body, err := a.httpapi.DoJSON(ctx, http.MethodPost, ManifestPath, req)
	if err != nil {
		return nil, err
	}

	var resp apitypesv1.SetManifestResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}
	return &resp, nil
}
