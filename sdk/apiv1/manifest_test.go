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

func TestSetManifest(t *testing.T) {
	withMinAPIVersion := func(version string) []byte {
		var m map[string]any
		require.NoError(t, json.Unmarshal(testManifest, &m))
		m["MinimumAPIVersion"] = version
		out, err := json.Marshal(m)
		require.NoError(t, err)
		return out
	}
	previousManifest := withMinAPIVersion("")
	previousTransition := history.BuildTransitionChain([][]byte{previousManifest})[0].Digest()
	nonce := make([]byte, apitypesv1.NonceSize)

	for name, tc := range map[string]struct {
		req               *apitypesv1.SetManifestRequest
		minimumAPIVersion string
		// history is the manifest history the Coordinator reports before the manifest of the request.
		history       [][]byte
		seedSharesDoc []byte
		// tamper modifies the Coordinator's response after it was attested.
		tamper      func(*apitypesv1.SetManifestResponse)
		apiErr      *apitypes.APIError
		validateErr error

		wantErr         string
		wantErrIs       error
		wantErrCode     apitypes.ErrorCode
		wantNoRequest   bool
		wantUnvalidated bool
	}{
		"initial manifest": {
			req:           &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: testManifest, Signature: []byte("signature")},
			seedSharesDoc: []byte(`{"salt":"c2FsdA=="}`),
		},
		"atomic update": {
			req:     &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: testManifest, PreviousTransitionHash: previousTransition[:]},
			history: [][]byte{previousManifest},
		},
		"update applied after another transition": {
			req:             &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: testManifest, PreviousTransitionHash: make([]byte, history.HashSize)},
			history:         [][]byte{previousManifest},
			wantErr:         "not after the requested",
			wantUnvalidated: true,
		},
		"coordinator reports another manifest": {
			req:             &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: testManifest},
			tamper:          func(r *apitypesv1.SetManifestResponse) { r.Manifests = [][]byte{previousManifest} },
			wantErr:         "isn't the one that was set",
			wantUnvalidated: true,
		},
		"coordinator reports no manifests": {
			req:             &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: testManifest},
			tamper:          func(r *apitypesv1.SetManifestResponse) { r.Manifests = nil },
			wantErr:         "does not include manifests",
			wantUnvalidated: true,
		},
		"failed validation": {
			req:         &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: testManifest},
			validateErr: assert.AnError,
			wantErr:     assert.AnError.Error(),
		},
		"coordinator rejects the request": {
			req: &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: testManifest},
			apiErr: &apitypes.APIError{
				StatusCode: http.StatusForbidden,
				Code:       apitypes.ErrorCodeInvalidSignature,
				Err:        "manifest signature is invalid",
			},
			wantErr:         "manifest signature is invalid",
			wantErrCode:     apitypes.ErrorCodeInvalidSignature,
			wantUnvalidated: true,
		},
		"manifest pins a newer API version": {
			req:           &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: withMinAPIVersion("v2")},
			wantErrIs:     apiversion.ErrMinimumUnmet,
			wantNoRequest: true,
		},
		"manifest pins v1": {
			req: &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: withMinAPIVersion(Version)},
		},
		"client pins a newer API version": {
			req:               &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: testManifest},
			minimumAPIVersion: "v2",
			wantErrIs:         apiversion.ErrMinimumUnmet,
			wantNoRequest:     true,
		},
		"client pins v1": {
			req:               &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: testManifest},
			minimumAPIVersion: Version,
		},
		"invalid manifest": {
			req:           &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: []byte("{")},
			wantErr:       "unmarshalling manifest",
			wantNoRequest: true,
		},
		"bad nonce": {
			req:           &apitypesv1.SetManifestRequest{Nonce: []byte{1, 2, 3}, Manifest: testManifest},
			wantErr:       "bad nonce length",
			wantNoRequest: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			coordinator := &fakeManifestCoordinator{
				history:       tc.history,
				seedSharesDoc: tc.seedSharesDoc,
				tamper:        tc.tamper,
				apiErr:        tc.apiErr,
			}
			srv := httptest.NewServer(coordinator)
			t.Cleanup(srv.Close)

			validator := &stubValidator{err: tc.validateErr}
			api := New(Config{
				HTTP: &httpapi.Client{HTTPClient: srv.Client(), BaseURL: srv.URL},
				Validator: func(m *manifest.Manifest) (validators.Validator, error) {
					// The reference values must come from the manifest being set.
					var want manifest.Manifest
					require.NoError(json.Unmarshal(tc.req.Manifest, &want))
					assert.Equal(&want, m)
					return validator, nil
				},
				MinimumAPIVersion: tc.minimumAPIVersion,
			})

			result, err := api.SetManifest(t.Context(), tc.req)
			assert.Equal(tc.wantNoRequest, coordinator.gotRequest == nil, "request sent")
			if tc.wantErr != "" || tc.wantErrIs != nil {
				require.Error(err)
				assert.ErrorContains(err, tc.wantErr)
				if tc.wantErrIs != nil {
					assert.ErrorIs(err, tc.wantErrIs)
				}
				if tc.wantErrCode != "" {
					var apiErr *apitypes.APIError
					require.ErrorAs(err, &apiErr)
					assert.Equal(tc.wantErrCode, apiErr.Code)
				}
				assert.Nil(result)
				if !tc.wantNoRequest {
					assert.Equal(tc.wantUnvalidated, validator.gotReportData == nil, "attestation validated")
				}
				return
			}
			require.NoError(err)

			assert.Equal(tc.req, coordinator.gotRequest)

			wantManifests := append(append([][]byte{}, tc.history...), tc.req.Manifest)
			transitions := history.BuildTransitionChain(wantManifests)
			wantTransition := transitions[len(transitions)-1].Digest()
			assert.Equal(wantManifests, result.Manifests)
			assert.Equal(wantTransition[:], result.LatestTransitionHash)
			assert.Equal([]byte("root-ca"), result.RootCA)
			assert.Equal([]byte("mesh-ca"), result.MeshCA)
			assert.Equal(tc.seedSharesDoc, result.SeedSharesDoc)

			wantReportData, err := apitypesv1.ConstructSetManifestReportData(tc.req.Nonce, wantTransition[:], coordinator.sentResponse)
			require.NoError(err)
			assert.Equal(wantReportData[:], validator.gotReportData)
		})
	}
}

// fakeManifestCoordinator serves the manifest endpoint by appending the requested manifest to its history.
type fakeManifestCoordinator struct {
	history       [][]byte
	seedSharesDoc []byte
	tamper        func(*apitypesv1.SetManifestResponse)
	apiErr        *apitypes.APIError

	gotRequest   *apitypesv1.SetManifestRequest
	sentResponse *apitypesv1.SetManifestResponse
}

func (f *fakeManifestCoordinator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != apitypesv1.ManifestPath || r.Method != apitypesv1.SetManifestMethod {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var req apitypesv1.SetManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	f.gotRequest = &req

	if f.apiErr != nil {
		w.WriteHeader(f.apiErr.StatusCode)
		if err := json.NewEncoder(w).Encode(f.apiErr); err != nil {
			panic(err)
		}
		return
	}

	resp := &apitypesv1.SetManifestResponse{
		AttestationType:   asn1.ObjectIdentifier{1, 2, 3},
		RawAttestationDoc: []byte("attestation"),
		SeedSharesDoc:     f.seedSharesDoc,
		CoordinatorState: apitypesv1.CoordinatorState{
			Manifests: append(append([][]byte{}, f.history...), req.Manifest),
			RootCA:    []byte("root-ca"),
			MeshCA:    []byte("mesh-ca"),
		},
	}
	if f.tamper != nil {
		f.tamper(resp)
	}
	f.sentResponse = resp
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		panic(err)
	}
}
