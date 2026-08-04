// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/sdk/apiv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetManifest(t *testing.T) {
	request := &apitypes.SetManifestRequest{
		Manifest:  []byte(`{"foo":"bar"}`),
		Policies:  [][]byte{[]byte("policy1")},
		Signature: []byte("signature"),
	}

	for name, tc := range map[string]struct {
		request             *apitypes.SetManifestRequest
		coordinatorVersions []string
		// pinned selects an explicitly versioned call instead of the negotiating one.
		pinned bool

		wantErr string
	}{
		"negotiated": {
			request:             request,
			coordinatorVersions: []string{apiv1.Version},
		},
		"pinned to v1": {
			request: request,
			pinned:  true,
		},
		"nil request": {
			coordinatorVersions: []string{apiv1.Version},
			wantErr:             "request must not be nil",
		},
		"no common API version": {
			request:             request,
			coordinatorVersions: []string{"v99"},
			wantErr:             "no common API version",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			srv := httptest.NewServer(coordinatorHandler(tc.coordinatorVersions))
			t.Cleanup(srv.Close)

			client := New().WithBaseURL(srv.URL)

			var resp *apitypes.SetManifestResponse
			var err error
			if tc.pinned {
				resp, err = client.V1().SetManifest(t.Context(), tc.request)
			} else {
				resp, err = client.SetManifest(t.Context(), tc.request)
			}

			if tc.wantErr != "" {
				require.Error(err)
				assert.Contains(err.Error(), tc.wantErr)
				assert.Nil(resp)
				return
			}

			require.NoError(err)
			require.NotNil(resp)
			assert.Equal(constants.Version, resp.Version)
			assert.Equal([]byte("root-ca"), resp.RootCA)
			assert.Equal([]byte("mesh-ca"), resp.MeshCA)
			require.NotNil(resp.SeedSharesDoc)
			require.Len(resp.SeedSharesDoc.SeedShares, 1)
			assert.Equal("public-key", resp.SeedSharesDoc.SeedShares[0].PublicKey)
		})
	}
}

// TestSetManifestRequestEncoding ensures the request reaches the Coordinator unchanged,
// at the versioned path.
func TestSetManifestRequestEncoding(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	want := &apitypes.SetManifestRequest{
		Manifest:               []byte(`{"foo":"bar"}`),
		Policies:               [][]byte{[]byte("policy1"), []byte("policy2")},
		PreviousTransitionHash: []byte("previous-transition-hash"),
		Signature:              []byte("signature"),
	}

	var got apitypes.SetManifestRequest
	var gotPath, gotMethod, gotContentType string
	mux := http.NewServeMux()
	mux.Handle(capabilitiesPath, capabilitiesHandler([]string{apiv1.Version}))
	mux.HandleFunc(apiv1.ManifestPath, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		gotPath, gotMethod, gotContentType = r.URL.Path, r.Method, r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		setManifestResponse(w)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	_, err := New().WithBaseURL(srv.URL).SetManifest(t.Context(), want)
	require.NoError(err)

	assert.Equal(apiv1.ManifestPath, gotPath)
	assert.Equal(http.MethodPost, gotMethod)
	assert.Equal("application/json", gotContentType)
	assert.Equal(*want, got)
}

// TestSetManifestBaseURLWithPathPrefix ensures a reverse-proxied base URL keeps its prefix.
func TestSetManifestBaseURLWithPathPrefix(t *testing.T) {
	require := require.New(t)

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		setManifestResponse(w)
	}))
	t.Cleanup(srv.Close)

	_, err := New().WithBaseURL(srv.URL+"/contrast").WithAPIVersion(apiv1.Version).
		SetManifest(t.Context(), &apitypes.SetManifestRequest{Manifest: []byte("{}")})
	require.NoError(err)
	require.Equal("/contrast"+apiv1.ManifestPath, gotPath)
}

// TestNoBaseURL ensures calls fail with a helpful error if no base URL was set.
func TestNoBaseURL(t *testing.T) {
	_, err := New().SetManifest(t.Context(), &apitypes.SetManifestRequest{})
	require.ErrorContains(t, err, "no base URL set")
}

// coordinatorHandler serves both the capabilities and the v1 manifest endpoint.
func coordinatorHandler(versions []string) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(capabilitiesPath, capabilitiesHandler(versions))
	mux.HandleFunc(apiv1.ManifestPath, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		setManifestResponse(w)
	})
	return mux
}

func setManifestResponse(w http.ResponseWriter) {
	resp := &apitypes.SetManifestResponse{
		Version: constants.Version,
		RootCA:  []byte("root-ca"),
		MeshCA:  []byte("mesh-ca"),
		SeedSharesDoc: &apitypes.SeedShareDocument{
			Salt: []byte("salt"),
			SeedShares: []apitypes.SeedShare{{
				PublicKey:     "public-key",
				EncryptedSeed: []byte("encrypted-seed"),
			}},
		},
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		panic(err)
	}
}
