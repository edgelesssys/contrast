// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apitypes

import (
	"crypto/sha256"
	"net/http"
)

// APIVersionV1 is the identifier of version 1 of the Contrast HTTP API.
const APIVersionV1 = "v1"

const (
	// CapabilitiesPath is the path of the capabilities endpoint.
	//
	// The endpoint is deliberately unversioned. It is how clients discover which versions exist,
	// so it must be reachable without knowing a version first.
	CapabilitiesPath = "/capabilities"
	// CapabilitiesMethod is the HTTP method of the capabilities endpoint.
	CapabilitiesMethod = http.MethodGet
)

// CapabilitiesResponse is the response body of the GET /capabilities endpoint.
//
// It tells clients which versions of the Contrast HTTP API the Coordinator supports,
// so they can decide whether, and at which version, to use the HTTP API instead of falling back to the gRPC API.
type CapabilitiesResponse struct {
	// APIVersions lists the HTTP API versions the Coordinator supports, e.g. ["v1"].
	APIVersions []string `json:"api_versions"`
}

// Digest returns a SHA-256 digest over the list of supported API versions.
func (c CapabilitiesResponse) Digest() [sha256.Size]byte {
	// digest = sha256(sha256(version[0]) || sha256(version[1]) || ...)
	var versionDigests []byte
	for _, version := range c.APIVersions {
		versionDigest := sha256.Sum256([]byte(version))
		versionDigests = append(versionDigests, versionDigest[:]...)
	}
	return sha256.Sum256(versionDigests)
}
