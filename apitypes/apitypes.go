// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package apitypes contains the wire-format types of the Contrast HTTP API.
//
// The types here define the request/response bodies exchanged with the Coordinator.
package apitypes

// Port is the listening port of the HTTP API server.
const Port = "1314"

// APIError is the body returned by the Contrast HTTP API if a request was not successful.
type APIError struct {
	// Version is the Coordinator version.
	Version    string `json:"version"`
	StatusCode int    `json:"status_code"`
	Err        string `json:"error"`
}
