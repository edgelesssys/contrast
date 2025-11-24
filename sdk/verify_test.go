// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/httpapi"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerify(t *testing.T) {
	tests := map[string]struct {
		expectedManifest []byte
		manifestHistory  [][]byte
		errMsg           string
	}{
		"Empty manifest history": {
			expectedManifest: []byte("expected"),
			manifestHistory:  [][]byte{},
			errMsg:           "manifest history is empty",
		},
		"Matching manifest": {
			expectedManifest: []byte("expected"),
			manifestHistory:  [][]byte{[]byte("old"), []byte("expected")},
		},
		"Non-matching manifest": {
			expectedManifest: []byte("expected"),
			manifestHistory:  [][]byte{[]byte("old"), []byte("current")},
			errMsg:           "active manifest does not match expected manifest",
		},
		"Matching manifest is not latest": {
			expectedManifest: []byte("expected"),
			manifestHistory:  [][]byte{[]byte("expected"), []byte("current")},
			errMsg:           "active manifest does not match expected manifest",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			client := Client{}
			err := client.Verify(tt.expectedManifest, tt.manifestHistory)

			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("actual error: '%v', expected error: '%v'", err, tt.errMsg)
			}
		})
	}
}

func attestationHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req httpapi.AttestationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp := &httpapi.AttestationResponse{
		Version: constants.Version,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Not much we can do here, since headers are already sent and we seem to be unable to
		// write. Since this is a test, panicking is probably fine. We could ignore the error, but
		// that makes the linter unhappy.
		panic(err)
	}
}

func badAttestationHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`{"version": "12345", "error": "my error"}`))
}

func TestGetAttestation(t *testing.T) {
	for name, tc := range map[string]struct {
		nonce     []byte
		getServer func(http.Handler) *httptest.Server
		handler   http.Handler
		wantErr   string
	}{
		"plain HTTP": {
			nonce:     make([]byte, 32),
			getServer: httptest.NewServer,
			handler:   http.HandlerFunc(attestationHandler),
		},
		"HTTPS": {
			nonce:     make([]byte, 32),
			getServer: httptest.NewTLSServer,
			handler:   http.HandlerFunc(attestationHandler),
		},
		"bad nonce": {
			handler:   http.HandlerFunc(attestationHandler),
			getServer: httptest.NewServer,
			wantErr:   "want 32",
		},
		"bad handler": {
			nonce:     make([]byte, 32),
			getServer: httptest.NewServer,
			handler:   http.HandlerFunc(badAttestationHandler),
			wantErr:   "my error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			srv := tc.getServer(tc.handler)
			t.Cleanup(srv.Close)

			client := New()
			if srv.TLS != nil {
				client.HTTPClient = srv.Client()
			}

			att, err := client.GetAttestation(t.Context(), srv.URL, tc.nonce)
			if tc.wantErr != "" {
				assert.ErrorContains(err, tc.wantErr)
				assert.Nil(att)
				return
			}
			assert.NoError(err)
			assert.NotNil(att)
		})
	}
}

func TestValidateAttestation(t *testing.T) {
	testNonce := make([]byte, 32)
	for name, tc := range map[string]struct {
		nonce       []byte
		resp        *httpapi.AttestationResponse
		validateErr error
		wantErr     string
	}{
		"success": {
			nonce: testNonce,
			resp: &httpapi.AttestationResponse{
				RawAttestationDoc: testNonce,
				CoordinatorState: httpapi.CoordinatorState{
					Manifests: [][]byte{testManifest},
				},
			},
		},
		"no manifests": {
			nonce: testNonce,
			resp: &httpapi.AttestationResponse{
				RawAttestationDoc: testNonce,
				CoordinatorState:  httpapi.CoordinatorState{},
			},
			wantErr: "coordinator state does not include manifests",
		},
		"bad nonce": {
			wantErr: "want 32",
		},
		"failed validation": {
			nonce: testNonce,
			resp: &httpapi.AttestationResponse{
				RawAttestationDoc: testNonce,
				CoordinatorState: httpapi.CoordinatorState{
					Manifests: [][]byte{testManifest},
				},
			},
			validateErr: assert.AnError,
			wantErr:     assert.AnError.Error(),
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			attestation, err := json.Marshal(tc.resp)
			require.NoError(err)

			c := New()
			c.validatorsFromManifestOverride = func(*certcache.CachedHTTPSGetter, *manifest.Manifest, *slog.Logger) ([]atls.Validator, error) {
				return []atls.Validator{&stubValidator{err: tc.validateErr}}, nil
			}
			state, err := c.ValidateAttestation(t.Context(), t.TempDir(), tc.nonce, attestation)
			if tc.wantErr != "" {
				assert.ErrorContains(err, tc.wantErr)
				assert.Nil(state)
				return
			}
			assert.NoError(err)
			assert.Equal(&tc.resp.CoordinatorState, state)
		})
	}
}

var testManifest = []byte(`
{
  "Policies": {
    "ef27c1c91a0ce044c67f0ec10d7c66ea9f178453dc96a233e97f0675578042f2": {
      "SANs": ["coordinator"],
      "WorkloadSecretID": "apps/v1/StatefulSet/default/coordinator",
      "Role": "coordinator"
    }
  },
  "ReferenceValues": {
    "snp": [
      {
        "MinimumTCB": {
          "BootloaderVersion": 3,
          "TEEVersion": 0,
          "SNPVersion": 23,
          "MicrocodeVersion": 213
        },
        "ProductName": "Milan",
        "TrustedMeasurement": "05c504736ca974b9ac0c84b5099f957907507c09e4844bd0672d0b647205f35837bd479ae35567b22b69ce636666c286",
        "GuestPolicy": {
          "ABIMinor": 0,
          "ABIMajor": 0,
          "SMT": true,
          "MigrateMA": false,
          "Debug": false,
          "SingleSocket": false,
          "CXLAllowed": false,
          "MemAES256XTS": false,
          "RAPLDis": false,
          "CipherTextHidingDRAM": false,
          "PageSwapDisable": false
        },
        "PlatformInfo": {
          "SMTEnabled": true,
          "TSMEEnabled": false,
          "ECCEnabled": false,
          "RAPLDisabled": false,
          "CiphertextHidingDRAMEnabled": false,
          "AliasCheckComplete": false,
          "TIOEnabled": false
        }
      }
	]
  }
}
`)

type stubValidator struct {
	atls.Validator

	err error
}

func (v *stubValidator) Validate(context.Context, []byte, []byte) error {
	return v.err
}
