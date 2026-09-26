// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIVersionGate(t *testing.T) {
	testCases := map[string]struct {
		endpointVersion int
		guard           *stubGuard

		expStatus int
		expCode   apitypes.ErrorCode
	}{
		"no pin allows v1": {
			endpointVersion: 1,
			guard:           &stubGuard{},
			expStatus:       http.StatusOK,
		},
		"no pin allows legacy": {
			endpointVersion: 0,
			guard:           &stubGuard{},
			expStatus:       http.StatusOK,
		},
		"pin v1 allows v1": {
			endpointVersion: 1,
			guard:           &stubGuard{minimumAPIVersion: "v1"},
			expStatus:       http.StatusOK,
		},
		"pin v1 rejects legacy": {
			endpointVersion: 0,
			guard:           &stubGuard{minimumAPIVersion: "v1"},
			expStatus:       http.StatusForbidden,
			expCode:         apitypes.ErrorCodeAPIVersionTooOld,
		},
		"pin v2 rejects v1": {
			endpointVersion: 1,
			guard:           &stubGuard{minimumAPIVersion: "v2"},
			expStatus:       http.StatusForbidden,
			expCode:         apitypes.ErrorCodeAPIVersionTooOld,
		},
		"no state passes through": {
			endpointVersion: 0,
			guard:           &stubGuard{getStateErr: stateguard.ErrNoState},
			expStatus:       http.StatusOK,
		},
		"stale state passes through": {
			endpointVersion: 0,
			guard:           &stubGuard{getStateErr: stateguard.ErrStaleState},
			expStatus:       http.StatusOK,
		},
		"unknown state error fails closed": {
			endpointVersion: 1,
			guard:           &stubGuard{getStateErr: assert.AnError},
			expStatus:       http.StatusInternalServerError,
			expCode:         apitypes.ErrorCodeInternal,
		},
		"unparseable pin fails closed": {
			endpointVersion: 1,
			guard:           &stubGuard{minimumAPIVersion: "not-a-version"},
			expStatus:       http.StatusInternalServerError,
			expCode:         apitypes.ErrorCodeInternal,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			gate := &APIVersionGate{
				Version:    tc.endpointVersion,
				StateGuard: tc.guard,
				Next: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
			}

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/attest", nil)
			rec := httptest.NewRecorder()

			gate.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			require.Equal(tc.expStatus, res.StatusCode)

			if tc.expCode != "" {
				var apiErr apitypes.APIError
				require.NoError(json.NewDecoder(res.Body).Decode(&apiErr))
				require.Equal(tc.expCode, apiErr.Code)
			}
		})
	}
}
