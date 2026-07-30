// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apitypes

// APIVersionV1 is the identifier of version 1 of the Contrast HTTP API.
const APIVersionV1 = "v1"

// CapabilitiesResponse is the response body of the GET /capabilities endpoint.
//
// It tells clients which versions of the Contrast HTTP API the Coordinator supports,
// so they can decide whether, and at which version, to use the HTTP API instead of falling back to the gRPC API.
type CapabilitiesResponse struct {
	// APIVersions lists the HTTP API versions the Coordinator supports, e.g. ["v1"].
	APIVersions []string `json:"apiVersions"`
}
