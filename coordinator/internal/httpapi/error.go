// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"errors"
	"net/http"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/edgelesssys/contrast/coordinator/internal/userapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// errorCode classifies err into the stable error code reported to HTTP API clients.
func errorCode(err error, statusCode int) apitypes.ErrorCode {
	switch {
	case errors.Is(err, userapi.ErrNoManifest),
		errors.Is(err, userapi.ErrNoStateToRecover):
		return apitypes.ErrorCodeNoManifest
	case errors.Is(err, userapi.ErrNeedsRecovery):
		return apitypes.ErrorCodeNeedsRecovery
	case errors.Is(err, userapi.ErrAlreadyRecovered):
		return apitypes.ErrorCodeAlreadyRecovered
	case errors.Is(err, userapi.ErrTransitionMismatch),
		errors.Is(err, stateguard.ErrConcurrentUpdate):
		return apitypes.ErrorCodeManifestConflict
	case errors.Is(err, userapi.ErrInvalidSignature):
		return apitypes.ErrorCodeInvalidSignature
	case errors.Is(err, errAPIVersionTooOld):
		return apitypes.ErrorCodeAPIVersionTooOld
	}

	switch {
	case statusCode == http.StatusForbidden:
		return apitypes.ErrorCodePermissionDenied
	case statusCode >= 400 && statusCode < 500:
		return apitypes.ErrorCodeInvalidArgument
	default:
		return apitypes.ErrorCodeInternal
	}
}

// errorMessage returns the message of err to report to HTTP API clients.
//
// Errors of the gRPC UserAPI, which some endpoints share their logic with, are stripped of their gRPC framing,
// as they didn't travel over gRPC.
func errorMessage(err error) string {
	if s, ok := status.FromError(err); ok {
		return s.Message()
	}
	return err.Error()
}

// httpStatusFromGRPC maps the gRPC status of an error of the UserAPI to the HTTP status code reported for it,
// as documented on the [apitypes.ErrorCode] the error maps to.
func httpStatusFromGRPC(err error) int {
	switch status.Code(err) {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	default:
		return http.StatusInternalServerError
	}
}
