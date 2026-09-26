// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"errors"
	"net/http"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/edgelesssys/contrast/coordinator/internal/userapi"
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
