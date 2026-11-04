// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/sdk"
	"github.com/stretchr/testify/require"
)

func TestGetCoordinatorStateHTTPMinimumAPIVersion(t *testing.T) {
	for name, tc := range map[string]struct {
		versions          []string
		minimumAPIVersion string
	}{
		"coordinator predates API versioning": {minimumAPIVersion: "v1"},
		"coordinator only supports older versions": {
			versions:          []string{},
			minimumAPIVersion: "v1",
		},
		"minimum above all common versions": {
			versions:          []string{"v1"},
			minimumAPIVersion: "v2",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			attestationRequested := false
			mux := http.NewServeMux()
			if tc.versions != nil {
				mux.HandleFunc(apitypes.CapabilitiesPath, func(w http.ResponseWriter, _ *http.Request) {
					_ = json.NewEncoder(w).Encode(apitypes.CapabilitiesResponse{APIVersions: tc.versions})
				})
			}
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != apitypes.CapabilitiesPath {
					attestationRequested = true
				}
				w.WriteHeader(http.StatusInternalServerError)
			})
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			_, err := getCoordinatorStateHTTP(t.Context(), t.TempDir(), srv.URL, []byte("{}"), "", tc.minimumAPIVersion, slog.New(slog.DiscardHandler))
			require.ErrorIs(err, sdk.ErrMinimumAPIVersionUnmet)
			require.False(attestationRequested, "no attestation must be requested if the minimum API version isn't met")
		})
	}
}
