// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"cmp"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/edgelesssys/contrast/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetManifestHTTP(t *testing.T) {
	key := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[0])
	ownedManifest := withWorkloadOwnerKey(t, testSetManifest, &key.PublicKey)
	previous := history.Digest([]byte("previous"))

	for name, tc := range map[string]struct {
		req               *httpSetRequest
		manifest          []byte
		versions          []string
		minimumAPIVersion string
		badReport         bool

		wantPrevious     [history.HashSize]byte
		wantSignature    []byte
		wantNoSignature  bool
		wantAttestation  bool
		wantNoSetRequest bool
		wantErrIs        error
		wantErr          string
	}{
		"initial manifest signed with listed key": {
			req:             &httpSetRequest{workloadOwnerKey: key},
			manifest:        ownedManifest,
			versions:        []string{apitypesv1.Version},
			wantAttestation: true,
		},
		"initial manifest not signed with unlisted key": {
			req:             &httpSetRequest{workloadOwnerKey: key},
			versions:        []string{apitypesv1.Version},
			wantNoSignature: true,
			wantAttestation: true,
		},
		"atomic update signed with key": {
			req:          &httpSetRequest{workloadOwnerKey: key, atomic: true, previousTransitionHash: previous[:]},
			versions:     []string{apitypesv1.Version},
			wantPrevious: previous,
		},
		"detached signature": {
			req:             &httpSetRequest{workloadOwnerKey: key, signature: []byte("detached")},
			manifest:        ownedManifest,
			versions:        []string{apitypesv1.Version},
			wantSignature:   []byte("detached"),
			wantAttestation: true,
		},
		"unsigned": {
			req:             &httpSetRequest{},
			versions:        []string{apitypesv1.Version},
			wantNoSignature: true,
			wantAttestation: true,
		},
		"coordinator fails validation before the initial manifest": {
			req:              &httpSetRequest{workloadOwnerKey: key, atomic: true, previousTransitionHash: make([]byte, history.HashSize)},
			manifest:         ownedManifest,
			versions:         []string{apitypesv1.Version},
			badReport:        true,
			wantAttestation:  true,
			wantNoSetRequest: true,
			wantErr:          "checking the Coordinator before setting the initial manifest",
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

			manifestBytes := tc.manifest
			if manifestBytes == nil {
				manifestBytes = testSetManifest
			}
			coordinator := &fakeSetCoordinator{t: t, versions: tc.versions, badReport: tc.badReport}
			srv := httptest.NewServer(coordinator)
			t.Cleanup(srv.Close)

			t.Setenv("XDG_CACHE_HOME", t.TempDir())
			tc.req.baseURL = srv.URL
			tc.req.manifest = manifestBytes
			tc.req.minimumAPIVersion = cmp.Or(tc.minimumAPIVersion, apitypesv1.Version)
			_, err := setManifestHTTP(t.Context(), io.Discard, tc.req, slog.New(slog.DiscardHandler))
			require.Error(err)
			assert.Equal(tc.wantAttestation, coordinator.attestations > 0, "attestation requested")
			if tc.wantNoSetRequest {
				if tc.wantErrIs != nil {
					require.ErrorIs(err, tc.wantErrIs)
				}
				require.ErrorContains(err, tc.wantErr)
				assert.Empty(coordinator.setRequests)
				return
			}
			var apiErr *apitypes.APIError
			require.ErrorAs(err, &apiErr)
			assert.Equal(apitypes.ErrorCodeInvalidSignature, apiErr.Code)

			require.Len(coordinator.setRequests, 1)
			gotSetRequest := coordinator.setRequests[0]
			assert.Equal(manifestBytes, gotSetRequest.Manifest)
			assert.Len(gotSetRequest.Nonce, apitypesv1.NonceSize)
			if tc.req.atomic {
				assert.Equal(tc.req.previousTransitionHash, gotSetRequest.PreviousTransitionHash)
			} else {
				// The fake Coordinator is uninitialized, so the latest transition is all zeroes.
				assert.Equal(make([]byte, history.HashSize), gotSetRequest.PreviousTransitionHash)
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

// testSetManifest is a minimal valid manifest for setting it at a fake Coordinator.
var testSetManifest = []byte(`{"Policies":{"ef27c1c91a0ce044c67f0ec10d7c66ea9f178453dc96a233e97f0675578042f2":{"SANs":["coordinator"],"WorkloadSecretID":"coordinator","Role":"coordinator"}},"ReferenceValues":{"snp":[{"Platform":"Metal-QEMU-Insecure"}]}}`)

// testSetManifestCoordinatorPolicy is the hash of the Coordinator policy in testSetManifest.
const testSetManifestCoordinatorPolicy = "ef27c1c91a0ce044c67f0ec10d7c66ea9f178453dc96a233e97f0675578042f2"

// withWorkloadOwnerKey returns manifestBytes with key added to its workload owner keys.
func withWorkloadOwnerKey(t *testing.T, manifestBytes []byte, key *ecdsa.PublicKey) []byte {
	t.Helper()
	var m manifest.Manifest
	require.NoError(t, json.Unmarshal(manifestBytes, &m))
	m.WorkloadOwnerPubKeys = append(m.WorkloadOwnerPubKeys, manifest.MarshalWorkloadOwnerPubKey(key))
	out, err := json.Marshal(m)
	require.NoError(t, err)
	return out
}

// fakeSetCoordinator is an uninitialized Coordinator of testSetManifest's insecure platform that rejects all requests to set a manifest.
type fakeSetCoordinator struct {
	t         *testing.T
	versions  []string
	badReport bool

	attestations int
	setRequests  []*apitypesv1.SetManifestRequest
}

func (f *fakeSetCoordinator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	capabilitiesDigest := apitypes.CapabilitiesResponse{APIVersions: f.versions}.Digest()
	switch r.URL.Path {
	case apitypes.CapabilitiesPath:
		assert.NoError(f.t, json.NewEncoder(w).Encode(apitypes.CapabilitiesResponse{APIVersions: f.versions}))
	case apitypesv1.AttestPath:
		f.attestations++
		var req apitypesv1.AttestationRequest
		if !assert.NoError(f.t, json.NewDecoder(r.Body).Decode(&req)) {
			return
		}
		resp := apitypesv1.AttestationResponse{
			Status:             apitypesv1.StatusUninitialized,
			AttestationType:    oid.RawInsecureReport,
			CapabilitiesDigest: capabilitiesDigest[:],
		}
		reportData, err := apitypesv1.ConstructAttestationReportData(req.Nonce, nil, &resp)
		if !assert.NoError(f.t, err) {
			return
		}
		if f.badReport {
			reportData[0] ^= 0xff
		}
		hostData, err := hex.DecodeString(testSetManifestCoordinatorPolicy)
		if !assert.NoError(f.t, err) {
			return
		}
		resp.RawAttestationDoc, err = json.Marshal(struct {
			ReportData []byte `json:"reportData"`
			HostData   []byte `json:"hostData"`
		}{ReportData: reportData[:], HostData: hostData})
		if !assert.NoError(f.t, err) {
			return
		}
		assert.NoError(f.t, json.NewEncoder(w).Encode(resp))
	case apitypesv1.ManifestPath:
		var req apitypesv1.SetManifestRequest
		assert.NoError(f.t, json.NewDecoder(r.Body).Decode(&req))
		f.setRequests = append(f.setRequests, &req)
		apiErr := apitypes.APIError{StatusCode: http.StatusForbidden, Code: apitypes.ErrorCodeInvalidSignature}
		w.WriteHeader(apiErr.StatusCode)
		assert.NoError(f.t, json.NewEncoder(w).Encode(apiErr))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}
