// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/sdk/apiv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNegotiateAPIVersion(t *testing.T) {
	for name, tc := range map[string]struct {
		coordinatorVersions []string
		handler             http.Handler

		wantVersion string
		wantErr     string
	}{
		"coordinator supports v1": {
			coordinatorVersions: []string{apiv1.Version},
			wantVersion:         apiv1.Version,
		},
		"coordinator supports a newer version, too": {
			coordinatorVersions: []string{apiv1.Version, "v2"},
			wantVersion:         apiv1.Version,
		},
		"no common version": {
			coordinatorVersions: []string{"v99"},
			wantErr:             "no common API version",
		},
		"coordinator has no capabilities endpoint": {
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}),
			wantErr: "getting capabilities",
		},
		"capabilities response is malformed": {
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("not json"))
			}),
			wantErr: "unmarshalling capabilities",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			handler := tc.handler
			if handler == nil {
				handler = capabilitiesHandler(tc.coordinatorVersions)
			}
			srv := httptest.NewServer(handler)
			t.Cleanup(srv.Close)

			version, err := New(srv.URL).NegotiateAPIVersion(t.Context())

			if tc.wantErr != "" {
				require.Error(err)
				assert.Contains(err.Error(), tc.wantErr)
				return
			}
			require.NoError(err)
			assert.Equal(tc.wantVersion, version)
		})
	}
}

// TestNegotiateAPIVersionCaching ensures the Coordinator is only asked once per Client.
func TestNegotiateAPIVersionCaching(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		capabilitiesHandler([]string{apiv1.Version}).ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	client := New(srv.URL)
	for range 3 {
		version, err := client.NegotiateAPIVersion(t.Context())
		require.NoError(err)
		assert.Equal(apiv1.Version, version)
	}
	assert.Equal(int32(1), calls.Load())
}

// TestNegotiateAPIVersionErrorNotCached ensures a transient failure doesn't poison the Client.
func TestNegotiateAPIVersionErrorNotCached(t *testing.T) {
	require := require.New(t)

	var fail atomic.Bool
	fail.Store(true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		capabilitiesHandler([]string{apiv1.Version}).ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	client := New(srv.URL)
	_, err := client.NegotiateAPIVersion(t.Context())
	require.Error(err)

	fail.Store(false)
	version, err := client.NegotiateAPIVersion(t.Context())
	require.NoError(err)
	require.Equal(apiv1.Version, version)
}

// TestWithAPIVersionSkipsNegotiation ensures a pinned version doesn't contact the Coordinator.
func TestWithAPIVersionSkipsNegotiation(t *testing.T) {
	require := require.New(t)

	var contacted atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		contacted.Store(true)
	}))
	t.Cleanup(srv.Close)

	version, err := New(srv.URL).WithAPIVersion(apiv1.Version).NegotiateAPIVersion(t.Context())
	require.NoError(err)
	require.Equal(apiv1.Version, version)
	require.False(contacted.Load(), "Coordinator must not be contacted for a pinned API version")
}

func capabilitiesHandler(versions []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != capabilitiesPath || r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(apitypes.CapabilitiesResponse{APIVersions: versions}); err != nil {
			panic(err)
		}
	})
}
