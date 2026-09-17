// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apitypes

import (
	"fmt"
	"net/http"
)

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
	// StatusCode is the HTTP status code of the response, repeated here so that
	// the body is self-contained when it is logged or passed on.
	StatusCode int `json:"status_code"`
	// Code identifies the failure. Clients should branch on this field.
	Code ErrorCode `json:"code"`
	// Err is a human-readable description of the failure.
	// Clients should not branch on this field. Instead, branch on the [ErrorCode].
	Err string `json:"error"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP API call failed with %d (%s): %s", e.StatusCode, http.StatusText(e.StatusCode), e.Err)
}

// ErrorCode is a stable identifier for a failure returned by the Contrast HTTP API.
//
// Codes are an append-only vocabulary shared by all API versions.
// Existing codes are never removed or repurposed.
//
// Clients must tolerate codes they don't know and fall back to the behaviour implied by the HTTP status code.
// An unknown code must not be read as success, nor as a permanent failure, without looking at the status.
type ErrorCode string

const (
	// ErrorCodeInvalidArgument means the request was malformed or the manifest was rejected.
	// Retrying the identical request will fail again.
	// HTTP 400.
	ErrorCodeInvalidArgument ErrorCode = "invalid_argument"

	// ErrorCodeInvalidSignature means the request was not signed by an authorized workload owner.
	// Retrying the identical request will fail again.
	// HTTP 403.
	ErrorCodeInvalidSignature ErrorCode = "invalid_signature"

	// ErrorCodePermissionDenied means the caller is not allowed to perform the request
	// for a reason other than a bad signature, e.g. an attempt to change the seedshare owners.
	// Retrying the identical request will fail again.
	// HTTP 403.
	ErrorCodePermissionDenied ErrorCode = "permission_denied"

	// ErrorCodeNoManifest means no manifest has been set yet, so there is no state to serve or recover.
	// Retrying makes sense only once a manifest has been set.
	// HTTP 412.
	ErrorCodeNoManifest ErrorCode = "no_manifest"

	// ErrorCodeNeedsRecovery means the Coordinator has persisted state but no secrets, e.g. after a restart.
	// Retrying makes sense only after recovery.
	// HTTP 412.
	ErrorCodeNeedsRecovery ErrorCode = "needs_recovery"

	// ErrorCodeAlreadyRecovered means recovery was requested but the Coordinator already holds its secrets.
	// HTTP 412.
	ErrorCodeAlreadyRecovered ErrorCode = "already_recovered"

	// ErrorCodeManifestConflict means the manifest was not applied because the
	// Coordinator's latest transition is not the one the request expected.
	//
	// Retry only after re-reading the current transition hash and re-signing the request.
	// Retrying the identical request will fail again.
	// HTTP 412.
	ErrorCodeManifestConflict ErrorCode = "manifest_conflict"

	// ErrorCodeAPIVersionTooOld means the deployment's manifest pins a minimum API version,
	// and the request used an older version or an unversioned legacy endpoint.
	// Retrying the identical request will fail again. Retry against a newer API version.
	// HTTP 403.
	ErrorCodeAPIVersionTooOld ErrorCode = "api_version_too_old"

	// ErrorCodeInternal means the Coordinator failed for a reason that is not the caller's fault.
	// Retrying is reasonable.
	// HTTP 5xx.
	ErrorCodeInternal ErrorCode = "internal"
)
