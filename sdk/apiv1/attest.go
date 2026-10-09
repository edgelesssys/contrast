// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package apiv1

import (
	"context"
	"fmt"
	"net/http"

	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/cryptohelpers"
)

// GetAttestation requests attestation evidence from the Coordinator.
//
// The nonce needs to be exactly 32 bytes, which should come from a CSPRNG.
func (a *API) GetAttestation(ctx context.Context, nonce []byte) ([]byte, error) {
	if len(nonce) != cryptohelpers.RNGLengthDefault {
		return nil, fmt.Errorf("bad nonce length: got %d, want %d", len(nonce), cryptohelpers.RNGLengthDefault)
	}
	return a.httpapi.DoJSON(ctx, http.MethodPost, apitypesv1.AttestPath, &apitypesv1.AttestationRequest{Nonce: nonce})
}
