// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/edgelesssys/contrast/apitypes"
)

// supportedAPIVersions are the HTTP API versions this Coordinator serves.
//
// Clients compare this against the versions they know and pick the newest shared one,
// or fall back to the gRPC API on error or no matching supported versions.
var supportedAPIVersions = []string{apitypes.APIVersionV1}

// CapabilitiesHandler handles GET requests to /capabilities.
// It advertises which versions of the Contrast HTTP API the Coordinator supports.
//
// This endpoint is deliberately NOT versioned, since it is used by clients to discover which versions exist.
// The response body must only ever be extended.
type CapabilitiesHandler struct{}

// ServeHTTP implements [http.Handler].
func (h *CapabilitiesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	resp := apitypes.CapabilitiesResponse{
		APIVersions: supportedAPIVersions,
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(resp); err != nil {
		log.Printf("encoding capabilities response: %v", err)
	}
}
