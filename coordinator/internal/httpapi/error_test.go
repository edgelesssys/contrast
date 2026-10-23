// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/edgelesssys/contrast/coordinator/internal/userapi"
	"github.com/stretchr/testify/assert"
)

func TestErrorCode(t *testing.T) {
	for name, tc := range map[string]struct {
		err        error
		statusCode int
		want       apitypes.ErrorCode
	}{
		"no manifest": {
			err:        userapi.ErrNoManifest,
			statusCode: http.StatusPreconditionFailed,
			want:       apitypes.ErrorCodeNoManifest,
		},
		"no state to recover": {
			err:        userapi.ErrNoStateToRecover,
			statusCode: http.StatusPreconditionFailed,
			want:       apitypes.ErrorCodeNoManifest,
		},
		"needs recovery": {
			err:        userapi.ErrNeedsRecovery,
			statusCode: http.StatusPreconditionFailed,
			want:       apitypes.ErrorCodeNeedsRecovery,
		},
		"already recovered": {
			err:        userapi.ErrAlreadyRecovered,
			statusCode: http.StatusPreconditionFailed,
			want:       apitypes.ErrorCodeAlreadyRecovered,
		},
		"transition mismatch": {
			err:        userapi.ErrTransitionMismatch,
			statusCode: http.StatusPreconditionFailed,
			want:       apitypes.ErrorCodeManifestConflict,
		},
		"concurrent update": {
			err:        stateguard.ErrConcurrentUpdate,
			statusCode: http.StatusPreconditionFailed,
			want:       apitypes.ErrorCodeManifestConflict,
		},
		"invalid signature": {
			err:        userapi.ErrInvalidSignature,
			statusCode: http.StatusForbidden,
			want:       apitypes.ErrorCodeInvalidSignature,
		},
		"wrapped sentinel": {
			err:        fmt.Errorf("updating Coordinator state: %w", stateguard.ErrConcurrentUpdate),
			statusCode: http.StatusPreconditionFailed,
			want:       apitypes.ErrorCodeManifestConflict,
		},
		"unclassified 403": {
			err:        errors.New("changes to seedshare owners are not allowed"),
			statusCode: http.StatusForbidden,
			want:       apitypes.ErrorCodePermissionDenied,
		},
		"unclassified 4xx": {
			err:        errors.New("unmarshalling manifest"),
			statusCode: http.StatusBadRequest,
			want:       apitypes.ErrorCodeInvalidArgument,
		},
		"unclassified 5xx": {
			err:        errors.New("getting state"),
			statusCode: http.StatusInternalServerError,
			want:       apitypes.ErrorCodeInternal,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, errorCode(tc.err, tc.statusCode))
		})
	}
}
