// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package apitypes contains the wire-format types of the Contrast HTTP API.
//
// The types here define the request/response bodies exchanged with the Coordinator.
package apitypes

import (
	"fmt"
	"net/http"
)

// Port is the listening port of the HTTP API server.
const Port = "1314"

// APIError is the body returned by the Contrast HTTP API if a request was not successful.
//
// It implements [error], so clients can inspect a failed API call with [errors.As]:
//
//	_, err := client.GetAttestation(ctx, nonce)
//	var apiErr *apitypes.APIError
//	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusPreconditionFailed {
//		// The Coordinator has no manifest yet.
//	}
type APIError struct {
	// Version is the Coordinator version.
	Version string `json:"version"`
	// StatusCode is the HTTP status code of the response.
	StatusCode int `json:"status_code"`
	// Err is the error message.
	Err string `json:"error"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP API call failed with %d (%s): %s", e.StatusCode, http.StatusText(e.StatusCode), e.Err)
}
