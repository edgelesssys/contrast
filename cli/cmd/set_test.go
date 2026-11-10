// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"cmp"
	"crypto/ecdsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/edgelesssys/contrast/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetManifestHTTP(t *testing.T) {
	key := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[0])
	manifestBytes := []byte(`{"Policies":{"ef27c1c91a0ce044c67f0ec10d7c66ea9f178453dc96a233e97f0675578042f2":{"SANs":["coordinator"],"WorkloadSecretID":"coordinator","Role":"coordinator"}},"ReferenceValues":{"snp":[{"Platform":"Metal-QEMU-Insecure"}]}}`)
	previous := history.Digest([]byte("previous"))

	for name, tc := range map[string]struct {
		req               *httpSetRequest
		versions          []string
		minimumAPIVersion string

		wantPrevious       [history.HashSize]byte
		wantSignature      []byte
		wantNoSignature    bool
		wantAttestation    bool
		wantNoSetRequest   bool
		wantErrIs          error
		wantInvalidSigCode bool
	}{
		"initial manifest signed with key": {
			req:             &httpSetRequest{workloadOwnerKey: key},
			versions:        []string{apitypesv1.Version},
			wantAttestation: true,
		},
		"atomic update signed with key": {
			req:          &httpSetRequest{workloadOwnerKey: key, atomic: true, previousTransitionHash: previous[:]},
			versions:     []string{apitypesv1.Version},
			wantPrevious: previous,
		},
		"detached signature": {
			req:           &httpSetRequest{workloadOwnerKey: key, signature: []byte("detached")},
			versions:      []string{apitypesv1.Version},
			wantSignature: []byte("detached"),
		},
		"unsigned": {
			req:             &httpSetRequest{},
			versions:        []string{apitypesv1.Version},
			wantNoSignature: true,
		},
		"minimum API version unmet": {
			req:               &httpSetRequest{workloadOwnerKey: key, atomic: true, previousTransitionHash: previous[:]},
			versions:          []string{apitypesv1.Version},
			minimumAPIVersion: "v2",
			wantNoSetRequest:  true,
			wantErrIs:         sdk.ErrMinimumAPIVersionUnmet,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			var gotAttestation bool
			var gotSetRequest *apitypesv1.SetManifestRequest
			mux := http.NewServeMux()
			mux.HandleFunc(apitypes.CapabilitiesPath, func(w http.ResponseWriter, _ *http.Request) {
				_ = json.NewEncoder(w).Encode(apitypes.CapabilitiesResponse{APIVersions: tc.versions})
			})
			mux.HandleFunc(apitypesv1.AttestPath, func(w http.ResponseWriter, _ *http.Request) {
				gotAttestation = true
				capabilitiesDigest := apitypes.CapabilitiesResponse{APIVersions: tc.versions}.Digest()
				_ = json.NewEncoder(w).Encode(apitypesv1.AttestationResponse{
					Status:             apitypesv1.StatusUninitialized,
					CapabilitiesDigest: capabilitiesDigest[:],
				})
			})
			mux.HandleFunc(apitypesv1.ManifestPath, func(w http.ResponseWriter, r *http.Request) {
				gotSetRequest = &apitypesv1.SetManifestRequest{}
				_ = json.NewDecoder(r.Body).Decode(gotSetRequest)
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(apitypes.APIError{StatusCode: http.StatusForbidden, Code: apitypes.ErrorCodeInvalidSignature})
			})
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			t.Setenv("XDG_CACHE_HOME", t.TempDir())
			tc.req.baseURL = srv.URL
			tc.req.manifest = manifestBytes
			tc.req.minimumAPIVersion = cmp.Or(tc.minimumAPIVersion, apitypesv1.Version)
			_, err := setManifestHTTP(t.Context(), io.Discard, tc.req, slog.New(slog.DiscardHandler))
			require.Error(err)
			assert.Equal(tc.wantAttestation, gotAttestation, "attestation requested")
			if tc.wantNoSetRequest {
				require.ErrorIs(err, tc.wantErrIs)
				assert.Nil(gotSetRequest)
				return
			}
			var apiErr *apitypes.APIError
			require.ErrorAs(err, &apiErr)
			assert.Equal(apitypes.ErrorCodeInvalidSignature, apiErr.Code)

			require.NotNil(gotSetRequest)
			assert.Equal(manifestBytes, gotSetRequest.Manifest)
			assert.Len(gotSetRequest.Nonce, apitypesv1.NonceSize)
			if tc.req.atomic {
				assert.Equal(tc.req.previousTransitionHash, gotSetRequest.PreviousTransitionHash)
			} else {
				assert.Empty(gotSetRequest.PreviousTransitionHash)
			}
			switch {
			case tc.wantNoSignature:
				assert.Empty(gotSetRequest.Signature)
			case tc.wantSignature != nil:
				assert.Equal(tc.wantSignature, gotSetRequest.Signature)
			default:
				tr := &history.Transition{ManifestHash: history.Digest(manifestBytes), PreviousTransitionHash: tc.wantPrevious}
				digest := tr.OwnerSigningDigest()
				assert.True(ecdsa.VerifyASN1(&key.PublicKey, digest[:], gotSetRequest.Signature), "signature must be valid for the expected transition")
			}
		})
	}
}
