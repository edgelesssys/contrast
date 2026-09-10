// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apitypes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIErrorWireFormat(t *testing.T) {
	data, err := json.Marshal(&APIError{
		Version:    "v1.2.3",
		StatusCode: 412,
		Code:       ErrorCodeNoManifest,
		Err:        "no manifest set",
	})
	require.NoError(t, err)

	assert.JSONEq(t, `{"version":"v1.2.3","status_code":412,"code":"no_manifest","error":"no manifest set"}`, string(data))
}

func TestErrorCodeValues(t *testing.T) {
	for code, want := range map[ErrorCode]string{
		ErrorCodeInvalidArgument:  "invalid_argument",
		ErrorCodeInvalidSignature: "invalid_signature",
		ErrorCodePermissionDenied: "permission_denied",
		ErrorCodeNoManifest:       "no_manifest",
		ErrorCodeNeedsRecovery:    "needs_recovery",
		ErrorCodeAlreadyRecovered: "already_recovered",
		ErrorCodeManifestConflict: "manifest_conflict",
		ErrorCodeInternal:         "internal",
	} {
		assert.Equal(t, want, string(code))
	}
}
