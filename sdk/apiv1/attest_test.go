// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package apiv1

import (
	"encoding/asn1"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/atls/validators"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/sdk/internal/apiversion"
	"github.com/edgelesssys/contrast/sdk/internal/httpapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAttestation(t *testing.T) {
	for name, tc := range map[string]struct {
		nonce             []byte
		minimumAPIVersion string

		wantErr   string
		wantErrIs error
	}{
		"success": {
			nonce: make([]byte, apitypesv1.NonceSize),
		},
		"client pins v1": {
			nonce:             make([]byte, apitypesv1.NonceSize),
			minimumAPIVersion: Version,
		},
		"client pins a newer API version": {
			nonce:             make([]byte, apitypesv1.NonceSize),
			minimumAPIVersion: "v2",
			wantErrIs:         apiversion.ErrMinimumUnmet,
		},
		"bad nonce": {
			nonce:   []byte{1, 2, 3},
			wantErr: "bad nonce length",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			var gotRequest *apitypesv1.AttestationRequest
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != apitypesv1.AttestPath || r.Method != apitypesv1.AttestMethod {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				gotRequest = &apitypesv1.AttestationRequest{}
				if err := json.NewDecoder(r.Body).Decode(gotRequest); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				_, _ = w.Write([]byte(`{"version":"test"}`))
			}))
			t.Cleanup(srv.Close)

			api := New(Config{
				HTTP:              &httpapi.Client{HTTPClient: srv.Client(), BaseURL: srv.URL},
				MinimumAPIVersion: func() string { return tc.minimumAPIVersion },
			})
			attestation, err := api.GetAttestation(t.Context(), tc.nonce)
			if tc.wantErr != "" || tc.wantErrIs != nil {
				require.Error(err)
				assert.ErrorContains(err, tc.wantErr)
				if tc.wantErrIs != nil {
					assert.ErrorIs(err, tc.wantErrIs)
				}
				assert.Nil(gotRequest, "no attestation must be requested")
				return
			}
			require.NoError(err)
			assert.JSONEq(`{"version":"test"}`, string(attestation))
			require.NotNil(gotRequest)
			assert.Equal(tc.nonce, gotRequest.Nonce)
		})
	}
}

func TestValidateAttestation(t *testing.T) {
	testNonce := make([]byte, apitypesv1.NonceSize)

	manifestWithMinAPIVersion := func(version string) []byte {
		var m map[string]any
		require.NoError(t, json.Unmarshal(testManifest, &m))
		m["MinimumAPIVersion"] = version
		out, err := json.Marshal(m)
		require.NoError(t, err)
		return out
	}
	capabilities := func(versions ...string) *apitypes.CapabilitiesResponse {
		return &apitypes.CapabilitiesResponse{APIVersions: versions}
	}
	digest := func(versions ...string) []byte {
		d := capabilities(versions...).Digest()
		return d[:]
	}
	// response returns the attestation of an active Coordinator.
	response := func(capabilitiesDigest []byte, manifests ...[]byte) *apitypesv1.AttestationResponse {
		resp := &apitypesv1.AttestationResponse{
			AttestationType:           asn1.ObjectIdentifier{1, 2, 3},
			RawAttestationDoc:         testNonce,
			CapabilitiesDigest:        capabilitiesDigest,
			Status:                    apitypesv1.StatusActive,
			HPKEPubKey:                []byte("hpke-pub-key"),
			LatestTransitionSignature: []byte("transition-signature"),
			CoordinatorState: apitypesv1.CoordinatorState{
				Manifests: manifests,
			},
		}
		if len(manifests) > 0 {
			resp.Policies = [][]byte{testPolicy}
		}
		return resp
	}
	withPolicies := func(policies [][]byte, resp *apitypesv1.AttestationResponse) *apitypesv1.AttestationResponse {
		resp.Policies = policies
		return resp
	}
	// withStatus sets the status, and drops the latest transition signature, which only an active Coordinator reports.
	withStatus := func(status apitypesv1.Status, resp *apitypesv1.AttestationResponse) *apitypesv1.AttestationResponse {
		resp.Status = status
		if status != apitypesv1.StatusActive {
			resp.LatestTransitionSignature = nil
		}
		return resp
	}
	withTransitionSignature := func(signature []byte, resp *apitypesv1.AttestationResponse) *apitypesv1.AttestationResponse {
		resp.LatestTransitionSignature = signature
		return resp
	}

	for name, tc := range map[string]struct {
		nonce []byte
		// negotiated are the capabilities the SDK client negotiated with, nil if it didn't negotiate.
		negotiated        *apitypes.CapabilitiesResponse
		minimumAPIVersion string
		resp              *apitypesv1.AttestationResponse
		validateErr       error

		wantErr   string
		wantErrIs error
		// wantUnvalidated is set if the error must be returned without the attestation having been validated.
		wantUnvalidated bool
	}{
		"success": {
			nonce:      testNonce,
			negotiated: capabilities(Version),
			resp:       response(digest(Version), testManifest),
		},
		"coordinator supports a newer version than the SDK": {
			nonce:      testNonce,
			negotiated: capabilities("v2", Version),
			resp:       response(digest("v2", Version), testManifest),
		},
		"attestation relayed to a client that didn't negotiate": {
			nonce: testNonce,
			resp:  response(digest("v2", Version), testManifest),
		},
		"negotiated with forged capabilities": {
			nonce:      testNonce,
			negotiated: capabilities(Version),
			resp:       response(digest("v2", Version), testManifest),
			wantErrIs:  apiversion.ErrDowngrade,
		},
		"no manifests": {
			nonce:           testNonce,
			resp:            response(digest(Version)),
			wantErr:         "coordinator state does not include manifests",
			wantUnvalidated: true,
		},
		"bad nonce": {
			resp:    response(digest(Version), testManifest),
			wantErr: "bad nonce length",
		},
		"failed validation": {
			nonce:       testNonce,
			resp:        response(digest(Version), testManifest),
			validateErr: assert.AnError,
			wantErr:     assert.AnError.Error(),
		},
		"attestation without capabilities digest": {
			nonce:           testNonce,
			resp:            response(nil, testManifest),
			wantErr:         "doesn't include a capabilities digest",
			wantUnvalidated: true,
		},
		"attestation without status": {
			nonce:           testNonce,
			resp:            withStatus("", response(digest(Version), testManifest)),
			wantErr:         "unknown Coordinator status",
			wantUnvalidated: true,
		},
		"attestation with unknown status": {
			nonce:           testNonce,
			resp:            withStatus("bogus", response(digest(Version), testManifest)),
			wantErr:         "unknown Coordinator status",
			wantUnvalidated: true,
		},
		"uninitialized coordinator": {
			nonce:           testNonce,
			resp:            withStatus(apitypesv1.StatusUninitialized, response(digest(Version))),
			wantErr:         "not validated",
			wantErrIs:       ErrCoordinatorUninitialized,
			wantUnvalidated: true,
		},
		"coordinator awaiting recovery": {
			nonce:           testNonce,
			resp:            withStatus(apitypesv1.StatusStale, response(digest(Version))),
			wantErr:         "not validated",
			wantErrIs:       ErrCoordinatorNeedsRecovery,
			wantUnvalidated: true,
		},
		"attestation without latest transition signature": {
			nonce:           testNonce,
			resp:            withTransitionSignature(nil, response(digest(Version), testManifest)),
			wantErr:         "doesn't include the latest transition signature",
			wantUnvalidated: true,
		},
		"uninitialized coordinator reporting a transition signature": {
			nonce:           testNonce,
			resp:            withTransitionSignature([]byte("transition-signature"), withStatus(apitypesv1.StatusUninitialized, response(digest(Version)))),
			wantErr:         "reports Coordinator state although",
			wantUnvalidated: true,
		},
		"uninitialized coordinator reporting state": {
			nonce:           testNonce,
			resp:            withStatus(apitypesv1.StatusUninitialized, response(digest(Version), testManifest)),
			wantErr:         "reports Coordinator state although",
			wantUnvalidated: true,
		},
		"manifest pins v1": {
			nonce: testNonce,
			resp:  response(digest(Version), manifestWithMinAPIVersion(Version)),
		},
		"manifest pins a newer API version": {
			nonce:     testNonce,
			resp:      response(digest(Version), manifestWithMinAPIVersion("v2")),
			wantErr:   "older than the minimum",
			wantErrIs: apiversion.ErrMinimumUnmet,
		},
		"client pins a newer API version": {
			nonce:             testNonce,
			minimumAPIVersion: "v2",
			resp:              response(digest(Version), testManifest),
			wantErr:           "older than the minimum",
			wantErrIs:         apiversion.ErrMinimumUnmet,
		},
		"policy missing": {
			nonce:   testNonce,
			resp:    withPolicies(nil, response(digest(Version), testManifest)),
			wantErr: "missing",
		},
		"policy not referenced by any manifest": {
			nonce:   testNonce,
			resp:    withPolicies([][]byte{testPolicy, otherPolicy}, response(digest(Version), testManifest)),
			wantErr: "isn't referenced by any manifest",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			attestation, err := json.Marshal(tc.resp)
			require.NoError(err)

			validator := &stubValidator{err: tc.validateErr}
			api := New(Config{
				HTTP: &httpapi.Client{},
				Validator: func(*manifest.Manifest) (validators.Validator, error) {
					return validator, nil
				},
				MinimumAPIVersion:      func() string { return tc.minimumAPIVersion },
				NegotiatedCapabilities: func() *apitypes.CapabilitiesResponse { return tc.negotiated },
			})
			state, err := api.ValidateAttestation(t.Context(), tc.nonce, attestation)
			if tc.wantErr != "" || tc.wantErrIs != nil {
				require.Error(err)
				assert.ErrorContains(err, tc.wantErr)
				if tc.wantErrIs != nil {
					assert.ErrorIs(err, tc.wantErrIs)
				}
				assert.Nil(state)
				if tc.nonce != nil {
					assert.Equal(tc.wantUnvalidated, validator.gotReportData == nil, "attestation validated")
				}
				return
			}
			require.NoError(err)

			transitions := history.BuildTransitionChain(tc.resp.Manifests)
			latestTransitionHash := transitions[len(transitions)-1].Digest()
			assert.Equal(&CoordinatorState{
				Manifests:                 tc.resp.Manifests,
				Policies:                  tc.resp.Policies,
				RootCA:                    tc.resp.RootCA,
				MeshCA:                    tc.resp.MeshCA,
				LatestTransitionHash:      latestTransitionHash[:],
				LatestTransitionSignature: tc.resp.LatestTransitionSignature,
			}, state)

			wantReportData, err := apitypesv1.ConstructAttestationReportData(tc.nonce, latestTransitionHash[:], tc.resp)
			require.NoError(err)
			assert.Equal(wantReportData[:], validator.gotReportData)
		})
	}
}
