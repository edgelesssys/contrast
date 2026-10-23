// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveURL(t *testing.T) {
	for name, tc := range map[string]struct {
		baseURL string
		path    string

		wantURL string
		wantErr string
	}{
		"plain base URL": {
			baseURL: "http://coordinator:1314",
			path:    "/v1/manifest",
			wantURL: "http://coordinator:1314/v1/manifest",
		},
		"trailing slash": {
			baseURL: "http://coordinator:1314/",
			path:    "/v1/manifest",
			wantURL: "http://coordinator:1314/v1/manifest",
		},
		"proxy prefix is kept": {
			baseURL: "https://proxy/contrast",
			path:    "/v1/manifest",
			wantURL: "https://proxy/contrast/v1/manifest",
		},
		"proxy prefix with trailing slash": {
			baseURL: "https://proxy/contrast/",
			path:    "/v1/manifest",
			wantURL: "https://proxy/contrast/v1/manifest",
		},
		"query is preserved": {
			baseURL: "http://coordinator:1314",
			path:    "/v1/manifest?generation=2",
			wantURL: "http://coordinator:1314/v1/manifest?generation=2",
		},
		"escaped path segments stay escaped": {
			baseURL: "http://coordinator:1314",
			path:    "/v1/foo%2Fbar",
			wantURL: "http://coordinator:1314/v1/foo%2Fbar",
		},
		"no base URL": {
			path:    "/v1/manifest",
			wantErr: "base URL is empty",
		},
		"missing scheme": {
			baseURL: "coordinator:1314",
			path:    "/v1/manifest",
			wantErr: "expected http(s)://host",
		},
		"unsupported scheme": {
			baseURL: "ftp://coordinator:1314",
			path:    "/v1/manifest",
			wantErr: "expected http(s)://host",
		},
		"no host": {
			baseURL: "http:///v1",
			path:    "/v1/manifest",
			wantErr: "expected http(s)://host",
		},
		"unparsable base URL": {
			baseURL: "http://coordinator:1314/%zz",
			path:    "/v1/manifest",
			wantErr: "parsing base URL",
		},
		"unparsable path": {
			baseURL: "http://coordinator:1314",
			path:    "/v1/%zz",
			wantErr: "parsing path",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			c := &Client{BaseURL: tc.baseURL}
			gotURL, err := c.resolveURL(tc.path)

			if tc.wantErr != "" {
				require.Error(err)
				assert.Contains(err.Error(), tc.wantErr)
				return
			}
			require.NoError(err)
			assert.Equal(tc.wantURL, gotURL)
		})
	}
}

// TestDoJSONError ensures failed requests return an inspectable [apitypes.APIError].
func TestDoJSONError(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		body   string

		wantErrMsg string
	}{
		"coordinator error body": {
			status:     http.StatusPreconditionFailed,
			body:       `{"version":"v1.2.3","status_code":412,"error":"no manifest set"}`,
			wantErrMsg: "no manifest set",
		},
		"valid JSON without an error message": {
			status:     http.StatusBadGateway,
			body:       `{}`,
			wantErrMsg: "502 Bad Gateway",
		},
		"non-JSON body": {
			status:     http.StatusInternalServerError,
			body:       "oops",
			wantErrMsg: "500 Internal Server Error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(srv.Close)

			c := &Client{HTTPClient: srv.Client(), BaseURL: srv.URL, Log: slog.New(slog.DiscardHandler)}
			_, err := c.DoJSON(t.Context(), http.MethodGet, "/capabilities", nil)

			require.Error(err)
			var apiErr *apitypes.APIError
			require.ErrorAs(err, &apiErr)
			// The response status wins over whatever the body claims.
			assert.Equal(tc.status, apiErr.StatusCode)
			assert.Equal(tc.wantErrMsg, apiErr.Err)
			assert.Contains(err.Error(), tc.wantErrMsg)
		})
	}
}

// TestDoJSONSendsBody ensures request bodies are JSON-encoded and typed as such.
func TestDoJSONSendsBody(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	var gotBody []byte
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"api_versions":["v1"]}`))
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTPClient: srv.Client(), BaseURL: srv.URL, Log: slog.New(slog.DiscardHandler)}
	resp, err := c.DoJSON(t.Context(), http.MethodPost, "/attest", &apitypesv1.AttestationRequest{Nonce: []byte("nonce")})
	require.NoError(err)

	assert.Equal("application/json", gotContentType)
	var req apitypesv1.AttestationRequest
	require.NoError(json.Unmarshal(gotBody, &req))
	assert.Equal([]byte("nonce"), req.Nonce)
	assert.JSONEq(`{"api_versions":["v1"]}`, string(resp))
}
