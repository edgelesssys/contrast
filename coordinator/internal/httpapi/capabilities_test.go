// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/stretchr/testify/require"
)

func TestCapabilitiesHandler(t *testing.T) {
	testCases := map[string]struct {
		method    string
		expStatus int
	}{
		"success": {
			method:    http.MethodGet,
			expStatus: http.StatusOK,
		},
		"wrong HTTP method": {
			method:    http.MethodPost,
			expStatus: http.StatusMethodNotAllowed,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			handler := &CapabilitiesHandler{}

			req := httptest.NewRequestWithContext(t.Context(), tc.method, "/capabilities", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			require.Equal(tc.expStatus, res.StatusCode)

			if tc.expStatus == http.StatusOK {
				require.Equal("application/json", res.Header.Get("Content-Type"))
				var resp apitypes.CapabilitiesResponse
				require.NoError(json.NewDecoder(res.Body).Decode(&resp))
				require.Contains(resp.APIVersions, apitypes.APIVersionV1)
			}
		})
	}
}
