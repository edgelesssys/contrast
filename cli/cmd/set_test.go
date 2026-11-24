// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"cmp"
	"context"
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
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/edgelesssys/contrast/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
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
			req:             &httpSetRequest{workloadOwnerKey: key, atomic: true, previousTransitionHash: previous[:]},
			versions:        []string{apitypesv1.Version},
			wantPrevious:    previous,
			wantAttestation: true,
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
			wantErr:          "checking the Coordinator before setting the manifest",
		},
		"coordinator fails validation before an atomic update": {
			req:              &httpSetRequest{workloadOwnerKey: key, atomic: true, previousTransitionHash: previous[:]},
			versions:         []string{apitypesv1.Version},
			badReport:        true,
			wantAttestation:  true,
			wantNoSetRequest: true,
			wantErr:          "checking the Coordinator before setting the manifest",
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

func TestSetManifestHTTPRetry(t *testing.T) {
	key := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[0])

	for name, tc := range map[string]struct {
		failFirst    func(w http.ResponseWriter)
		wantAttempts int
	}{
		"lost response": {
			failFirst: func(w http.ResponseWriter) {
				conn, _, err := http.NewResponseController(w).Hijack()
				if err == nil {
					conn.Close()
				}
			},
			wantAttempts: 2,
		},
		"response interrupted": {
			failFirst: func(w http.ResponseWriter) {
				// Promise more than is sent, so that the client sees the connection break mid-body.
				w.Header().Set("Content-Length", "100")
				_, _ = w.Write([]byte(`{"version":`))
			},
			wantAttempts: 2,
		},
		"internal error": {
			failFirst: func(w http.ResponseWriter) {
				writeAPIError(t, w, apitypes.APIError{StatusCode: http.StatusInternalServerError, Code: apitypes.ErrorCodeInternal})
			},
			wantAttempts: 2,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			// The final attempt fails, too, as the fake can't attest a successful response.
			coordinator := &fakeSetCoordinator{t: t, versions: []string{apitypesv1.Version}, failFirst: tc.failFirst}
			srv := httptest.NewServer(coordinator)
			t.Cleanup(srv.Close)

			t.Setenv("XDG_CACHE_HOME", t.TempDir())
			_, err := setManifestHTTP(t.Context(), io.Discard, &httpSetRequest{
				baseURL:           srv.URL,
				minimumAPIVersion: apitypesv1.Version,
				manifest:          withWorkloadOwnerKey(t, testSetManifest, &key.PublicKey),
				workloadOwnerKey:  key,
			}, slog.New(slog.DiscardHandler))
			require.Error(err)

			require.Len(coordinator.setRequests, tc.wantAttempts)
			// The latest transition is only fetched once. The SDK checks the uninitialized Coordinator before each attempt.
			assert.Equal(1+tc.wantAttempts, coordinator.attestations, "attestations")
			setRequests := coordinator.setRequests
			for _, req := range setRequests[1:] {
				assert.Equal(setRequests[0].Manifest, req.Manifest)
				assert.Equal(setRequests[0].PreviousTransitionHash, req.PreviousTransitionHash)
				assert.Equal(setRequests[0].Signature, req.Signature, "the request must not be signed anew")
				assert.NotEqual(setRequests[0].Nonce, req.Nonce)
			}
		})
	}
}

func TestGetLatestTransitionGRPC(t *testing.T) {
	manifests := [][]byte{[]byte("first"), []byte("second")}
	transitions := history.BuildTransitionChain(manifests)
	chainLatest := transitions[len(transitions)-1].Digest()
	reported := history.Digest([]byte("reported"))

	for name, tc := range map[string]struct {
		resp    *userapi.GetManifestsResponse
		err     error
		want    []byte
		wantErr bool
	}{
		"no manifest": {
			err:  status.Error(codes.FailedPrecondition, userapi.ErrNoManifest.Error()),
			want: make([]byte, history.HashSize),
		},
		"latest transition reported": {
			resp: &userapi.GetManifestsResponse{Manifests: manifests, LatestTransition: &userapi.LatestTransition{TransitionHash: reported[:]}},
			want: reported[:],
		},
		"latest transition not reported": {
			resp: &userapi.GetManifestsResponse{Manifests: manifests},
			want: chainLatest[:],
		},
		"needs recovery": {
			err:     status.Error(codes.FailedPrecondition, "coordinator is in recovery mode"),
			wantErr: true,
		},
		"nothing reported": {
			resp:    &userapi.GetManifestsResponse{},
			wantErr: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := getLatestTransitionGRPC(t.Context(), &stubUserAPIClient{getManifestsResp: tc.resp, getManifestsErr: tc.err})
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSetLoopRetry(t *testing.T) {
	require := require.New(t)

	latest := history.Digest([]byte("latest"))
	client := &stubUserAPIClient{
		getManifestsResp: &userapi.GetManifestsResponse{LatestTransition: &userapi.LatestTransition{TransitionHash: latest[:]}},
		setManifestErrs:  []error{status.Error(codes.Unavailable, "connection reset")},
		setManifestResp:  &userapi.SetManifestResponse{RootCA: []byte("root-ca")},
	}
	resp, err := setLoop(t.Context(), client, io.Discard, &userapi.SetManifestRequest{Manifest: []byte("manifest"), Signature: []byte("signature")})
	require.NoError(err)
	require.Equal([]byte("root-ca"), resp.RootCA)

	require.Equal(1, client.getManifestsCalls, "the latest transition must only be fetched once")
	require.Len(client.setManifestReqs, 2)
	for _, req := range client.setManifestReqs {
		require.Equal(latest[:], req.PreviousTransitionHash)
		require.Equal([]byte("manifest"), req.Manifest)
		require.Equal([]byte("signature"), req.Signature)
	}
}

type stubUserAPIClient struct {
	userapi.UserAPIClient

	getManifestsResp  *userapi.GetManifestsResponse
	getManifestsErr   error
	getManifestsCalls int

	setManifestErrs []error
	setManifestResp *userapi.SetManifestResponse
	setManifestReqs []*userapi.SetManifestRequest
}

func (s *stubUserAPIClient) GetManifests(context.Context, *userapi.GetManifestsRequest, ...grpc.CallOption) (*userapi.GetManifestsResponse, error) {
	s.getManifestsCalls++
	return s.getManifestsResp, s.getManifestsErr
}

func (s *stubUserAPIClient) SetManifest(_ context.Context, req *userapi.SetManifestRequest, _ ...grpc.CallOption) (*userapi.SetManifestResponse, error) {
	s.setManifestReqs = append(s.setManifestReqs, proto.CloneOf(req))
	if len(s.setManifestReqs) <= len(s.setManifestErrs) {
		return nil, s.setManifestErrs[len(s.setManifestReqs)-1]
	}
	return s.setManifestResp, nil
}

func writeAPIError(t *testing.T, w http.ResponseWriter, apiErr apitypes.APIError) {
	w.WriteHeader(apiErr.StatusCode)
	assert.NoError(t, json.NewEncoder(w).Encode(apiErr))
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
	// failFirst, if set, answers the first request to set a manifest.
	failFirst func(w http.ResponseWriter)

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
		if f.failFirst != nil && len(f.setRequests) == 1 {
			f.failFirst(w)
			return
		}
		apiErr := apitypes.APIError{StatusCode: http.StatusForbidden, Code: apitypes.ErrorCodeInvalidSignature}
		w.WriteHeader(apiErr.StatusCode)
		assert.NoError(f.t, json.NewEncoder(w).Encode(apiErr))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}
