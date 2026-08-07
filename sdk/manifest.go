// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"context"
	"fmt"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/sdk/apiv1"
)

// SetManifest sets a manifest at the Coordinator and returns the deployment's CA certificates.
//
// It speaks the newest API version supported by both this SDK and the Coordinator, determined
// once per Client via [Client.NegotiateAPIVersion]. To pin a version instead, call the
// corresponding versioned API, e.g. c.V1().SetManifest.
//
// The returned SeedSharesDoc is only set if this call set the initial manifest. It holds the
// secret seed, encrypted for each seedshare owner, and is required to recover the deployment.
//
// Note: this function does not verify that the Coordinator is trustworthy! Callers should
// verify it via [Client.GetAttestation] and [Client.ValidateAttestation] before relying on the
// response.
func (c Client) SetManifest(ctx context.Context, req *apitypes.SetManifestRequest) (*apitypes.SetManifestResponse, error) {
	version, err := c.NegotiateAPIVersion(ctx)
	if err != nil {
		return nil, err
	}

	switch version {
	case apiv1.Version:
		return c.V1().SetManifest(ctx, req)
	default:
		return nil, fmt.Errorf("SetManifest is not implemented for API version %q", version)
	}
}
