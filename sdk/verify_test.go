// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/atls/validators"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/sdk/apiv1"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attestationHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req apitypesv1.AttestationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp := &apitypesv1.AttestationResponse{
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

			client := New(srv.URL).
				WithFSStore(afero.NewBasePathFs(afero.NewOsFs(), t.TempDir()))

			if srv.TLS != nil {
				client = client.WithHTTPClient(srv.Client())
			}

			att, err := client.GetAttestation(t.Context(), tc.nonce)
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
	testOID := asn1.ObjectIdentifier{1, 2, 3}

	manifestWithMinAPIVersion := func(version string) []byte {
		var m map[string]any
		require.NoError(t, json.Unmarshal(testManifest, &m))
		m["MinimumAPIVersion"] = version
		out, err := json.Marshal(m)
		require.NoError(t, err)
		return out
	}

	for name, tc := range map[string]struct {
		nonce       []byte
		versions    []string
		resp        *apitypesv1.AttestationResponse
		validateErr error
		wantErr     string
	}{
		"success": {
			nonce:    testNonce,
			versions: []string{apiv1.Version},
			resp: &apitypesv1.AttestationResponse{
				AttestationType:   testOID,
				RawAttestationDoc: testNonce,
				CoordinatorState: apitypesv1.CoordinatorState{
					Manifests: [][]byte{testManifest},
				},
			},
		},
		"success on legacy endpoint": {
			nonce: testNonce,
			resp: &apitypesv1.AttestationResponse{
				AttestationType:   testOID,
				RawAttestationDoc: testNonce,
				CoordinatorState: apitypesv1.CoordinatorState{
					Manifests: [][]byte{testManifest},
				},
			},
		},
		"no manifests": {
			nonce:    testNonce,
			versions: []string{apiv1.Version},
			resp: &apitypesv1.AttestationResponse{
				AttestationType:   testOID,
				RawAttestationDoc: testNonce,
				CoordinatorState:  apitypesv1.CoordinatorState{},
			},
			wantErr: "coordinator state does not include manifests",
		},
		"bad nonce": {
			wantErr: "want 32",
		},
		"failed validation": {
			nonce:    testNonce,
			versions: []string{apiv1.Version},
			resp: &apitypesv1.AttestationResponse{
				AttestationType:   testOID,
				RawAttestationDoc: testNonce,
				CoordinatorState: apitypesv1.CoordinatorState{
					Manifests: [][]byte{testManifest},
				},
			},
			validateErr: assert.AnError,
			wantErr:     assert.AnError.Error(),
		},
		"manifest pins a newer API version": {
			nonce:    testNonce,
			versions: []string{apiv1.Version},
			resp: &apitypesv1.AttestationResponse{
				AttestationType:   testOID,
				RawAttestationDoc: testNonce,
				CoordinatorState: apitypesv1.CoordinatorState{
					Manifests: [][]byte{manifestWithMinAPIVersion("v2")},
				},
			},
			wantErr: "older than the minimum",
		},
		"manifest pins an API version, but the legacy endpoint was used": {
			nonce: testNonce,
			resp: &apitypesv1.AttestationResponse{
				AttestationType:   testOID,
				RawAttestationDoc: testNonce,
				CoordinatorState: apitypesv1.CoordinatorState{
					Manifests: [][]byte{manifestWithMinAPIVersion("v1")},
				},
			},
			wantErr: "older than the minimum",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			attestation, err := json.Marshal(tc.resp)
			require.NoError(err)

			srv := httptest.NewServer(coordinatorHandler(tc.versions, nil))
			t.Cleanup(srv.Close)
			c := New(srv.URL)
			// Fetch an attestation to make the Client settle on an endpoint. The stub's response is replaced below.
			_, err = c.GetAttestation(t.Context(), testNonce)
			require.NoError(err)

			validator := &stubValidator{err: tc.validateErr}
			c.validatorsFromManifestOverride = func(*certcache.CachedHTTPSGetter, *manifest.Manifest, *slog.Logger) (validators.Validator, error) {
				return validator, nil
			}
			state, err := c.ValidateAttestation(t.Context(), tc.nonce, attestation)
			if tc.wantErr != "" {
				assert.ErrorContains(err, tc.wantErr)
				assert.Nil(state)
				return
			}
			assert.NoError(err)

			transitions := history.BuildTransitionChain(tc.resp.Manifests)
			latestTransitionHash := transitions[len(transitions)-1].Digest()
			expected := &CoordinatorState{
				Manifests: tc.resp.Manifests,
				Policies:  tc.resp.Policies,
				RootCA:    tc.resp.RootCA,
				MeshCA:    tc.resp.MeshCA,
			}

			assert.Equal(expected, state)

			wantReportData := apitypesv1.ConstructReportData(tc.nonce, latestTransitionHash[:], &tc.resp.CoordinatorState)
			if tc.versions != nil {
				var capsBody bytes.Buffer
				require.NoError(json.NewEncoder(&capsBody).Encode(apitypes.CapabilitiesResponse{APIVersions: tc.versions}))
				capsDigest := sha256.Sum256(capsBody.Bytes())
				wantReportData = apitypesv1.ConstructReportDataWithCapabilities(tc.nonce, latestTransitionHash[:], capsDigest[:], &tc.resp.CoordinatorState)
			}
			assert.Equal(wantReportData[:], validator.gotReportData)
		})
	}
}

// TestGetAttestationEndpoint ensures the attestation is fetched from the newest endpoint both sides support.
func TestGetAttestationEndpoint(t *testing.T) {
	for name, tc := range map[string]struct {
		versions          []string
		minimumAPIVersion string

		wantPath  string
		wantErrIs error
	}{
		"coordinator supports v1": {
			versions: []string{apiv1.Version},
			wantPath: apitypesv1.AttestPath,
		},
		"coordinator predates API versioning": {
			wantPath: apitypesv1.LegacyAttestPath,
		},
		"no common version": {
			versions: []string{"v99"},
			wantPath: apitypesv1.LegacyAttestPath,
		},
		"minimum met": {
			versions:          []string{apiv1.Version},
			minimumAPIVersion: "v1",
			wantPath:          apitypesv1.AttestPath,
		},
		"minimum above all common versions": {
			versions:          []string{apiv1.Version},
			minimumAPIVersion: "v2",
			wantErrIs:         ErrMinimumAPIVersionUnmet,
		},
		"minimum pinned, but coordinator predates API versioning": {
			minimumAPIVersion: "v1",
			wantErrIs:         ErrMinimumAPIVersionUnmet,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			var gotPath string
			srv := httptest.NewServer(coordinatorHandler(tc.versions, &gotPath))
			t.Cleanup(srv.Close)

			client := New(srv.URL).WithExpectedManifest(&manifest.Manifest{MinimumAPIVersion: tc.minimumAPIVersion})
			_, err := client.GetAttestation(t.Context(), make([]byte, 32))
			if tc.wantErrIs != nil {
				require.ErrorIs(err, tc.wantErrIs)
				require.Empty(gotPath, "no attestation must be requested if the minimum API version isn't met")
				return
			}
			require.NoError(err)
			require.Equal(tc.wantPath, gotPath)
		})
	}
}

// coordinatorHandler stubs a Coordinator advertising the given API versions.
// Without versions, it stubs a Coordinator that predates API versioning.
func coordinatorHandler(versions []string, gotAttestPath *string) http.Handler {
	mux := http.NewServeMux()
	if versions != nil {
		mux.Handle(capabilitiesPath, capabilitiesHandler(versions))
	}
	for _, path := range []string{apitypesv1.LegacyAttestPath, apitypesv1.AttestPath} {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if gotAttestPath != nil {
				*gotAttestPath = r.URL.Path
			}
			attestationHandler(w, r)
		})
	}
	return mux
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
        "APEIP": "0080b004",
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
	err           error
	gotReportData []byte
}

func (v *stubValidator) Validate(_ context.Context, _ asn1.ObjectIdentifier, _ []byte, reportData []byte) error {
	v.gotReportData = reportData
	return v.err
}

func (v *stubValidator) String() string {
	return "stubValidator"
}
