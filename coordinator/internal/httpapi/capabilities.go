// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
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
type CapabilitiesHandler struct {
	body []byte
}

// NewCapabilitiesHandler returns a [CapabilitiesHandler] advertising the supported API versions.
func NewCapabilitiesHandler() *CapabilitiesHandler {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(apitypes.CapabilitiesResponse{APIVersions: supportedAPIVersions}); err != nil {
		// The response is a fixed struct of strings; failing to encode it is a programming error.
		panic(err)
	}
	return &CapabilitiesHandler{body: buf.Bytes()}
}

// Digest returns the SHA-256 digest of the exact response body this handler serves.
func (h *CapabilitiesHandler) Digest() [32]byte {
	return sha256.Sum256(h.body)
}

// ServeHTTP implements [http.Handler].
func (h *CapabilitiesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(h.body)
}
