// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package apiv1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/internal/cryptohelpers"
)

// AttestPath is the path of the attestation endpoint, relative to the API's base URL.
const AttestPath = "/v1/attest"

// GetAttestation requests attestation evidence from the Coordinator.
//
// The nonce needs to be exactly 32 bytes, which should come from a CSPRNG.
func (a *API) GetAttestation(ctx context.Context, nonce []byte) ([]byte, error) {
	if len(nonce) != cryptohelpers.RNGLengthDefault {
		return nil, fmt.Errorf("bad nonce length: got %d, want %d", len(nonce), cryptohelpers.RNGLengthDefault)
	}
	return a.transport.DoJSON(ctx, http.MethodPost, AttestPath, &apitypes.AttestationRequest{Nonce: nonce})
}
