// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/asn1"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	userapiserver "github.com/edgelesssys/contrast/coordinator/internal/userapi"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/history/aferostore"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestSetManifestHandler sets manifests through the handler backed by the actual UserAPI and state guard,
// and checks that clients can reproduce the report data from the response.
func TestSetManifestHandler(t *testing.T) {
	ownerKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[0])
	otherKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[1])
	manifestWithOwner := func(key *ecdsa.PrivateKey, minimumAPIVersion string) []byte {
		m := &manifest.Manifest{MinimumAPIVersion: minimumAPIVersion}
		if key != nil {
			m.WorkloadOwnerPubKeys = []manifest.HexString{manifest.MarshalWorkloadOwnerPubKey(&key.PublicKey)}
		}
		b, err := json.Marshal(m)
		require.NoError(t, err)
		return b
	}
	first := manifestWithOwner(ownerKey, "")
	second := manifestWithOwner(ownerKey, apitypesv1.Version)
	firstTransition := (&history.Transition{ManifestHash: history.Digest(first)}).Digest()
	secondTransition := (&history.Transition{ManifestHash: history.Digest(second), PreviousTransitionHash: firstTransition}).Digest()
	sign := func(key *ecdsa.PrivateKey, tr *history.Transition) []byte {
		digest := tr.OwnerSigningDigest()
		sig, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
		require.NoError(t, err)
		return sig
	}

	for name, tc := range map[string]struct {
		setup []*apitypesv1.SetManifestRequest
		req   *apitypesv1.SetManifestRequest

		wantHTTPStatus     int
		wantCode           apitypes.ErrorCode
		wantManifests      [][]byte
		wantTransition     [history.HashSize]byte
		wantSeedSharesDoc  bool
		wantErrNotContains string
	}{
		"initial manifest": {
			req: &apitypesv1.SetManifestRequest{
				Nonce:     nonce,
				Manifest:  first,
				Signature: sign(ownerKey, &history.Transition{ManifestHash: history.Digest(first)}),
			},
			wantHTTPStatus:    http.StatusOK,
			wantManifests:     [][]byte{first},
			wantTransition:    firstTransition,
			wantSeedSharesDoc: true,
		},
		"atomic initial manifest": {
			req: &apitypesv1.SetManifestRequest{
				Nonce:                  nonce,
				Manifest:               first,
				PreviousTransitionHash: make([]byte, history.HashSize),
				Signature:              sign(ownerKey, &history.Transition{ManifestHash: history.Digest(first)}),
			},
			wantHTTPStatus:    http.StatusOK,
			wantManifests:     [][]byte{first},
			wantTransition:    firstTransition,
			wantSeedSharesDoc: true,
		},
		"initial manifest without owner keys and signature": {
			req: &apitypesv1.SetManifestRequest{
				Nonce:    nonce,
				Manifest: manifestWithOwner(nil, ""),
			},
			wantHTTPStatus:    http.StatusOK,
			wantManifests:     [][]byte{manifestWithOwner(nil, "")},
			wantTransition:    (&history.Transition{ManifestHash: history.Digest(manifestWithOwner(nil, ""))}).Digest(),
			wantSeedSharesDoc: true,
		},
		"initial manifest without signature": {
			req:            &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: first},
			wantHTTPStatus: http.StatusForbidden,
			wantCode:       apitypes.ErrorCodeInvalidSignature,
			// The message must not carry the gRPC framing.
			wantErrNotContains: "rpc error",
		},
		"update": {
			setup: []*apitypesv1.SetManifestRequest{
				{Manifest: first, Signature: sign(ownerKey, &history.Transition{ManifestHash: history.Digest(first)})},
			},
			req: &apitypesv1.SetManifestRequest{
				Nonce:                  nonce,
				Manifest:               second,
				PreviousTransitionHash: firstTransition[:],
				Signature:              sign(ownerKey, &history.Transition{ManifestHash: history.Digest(second), PreviousTransitionHash: firstTransition}),
			},
			wantHTTPStatus: http.StatusOK,
			wantManifests:  [][]byte{first, second},
			wantTransition: secondTransition,
		},
		"update past a pinned API version": {
			setup: []*apitypesv1.SetManifestRequest{
				{Manifest: first, Signature: sign(ownerKey, &history.Transition{ManifestHash: history.Digest(first)})},
				{Manifest: second, Signature: sign(ownerKey, &history.Transition{ManifestHash: history.Digest(second), PreviousTransitionHash: firstTransition})},
			},
			req: &apitypesv1.SetManifestRequest{
				Nonce:     nonce,
				Manifest:  second,
				Signature: sign(ownerKey, &history.Transition{ManifestHash: history.Digest(second), PreviousTransitionHash: secondTransition}),
			},
			wantHTTPStatus: http.StatusOK,
			wantManifests:  [][]byte{first, second, second},
			wantTransition: (&history.Transition{ManifestHash: history.Digest(second), PreviousTransitionHash: secondTransition}).Digest(),
		},
		"update signed by another key": {
			setup: []*apitypesv1.SetManifestRequest{
				{Manifest: first, Signature: sign(ownerKey, &history.Transition{ManifestHash: history.Digest(first)})},
			},
			req: &apitypesv1.SetManifestRequest{
				Nonce:     nonce,
				Manifest:  second,
				Signature: sign(otherKey, &history.Transition{ManifestHash: history.Digest(second), PreviousTransitionHash: firstTransition}),
			},
			wantHTTPStatus: http.StatusForbidden,
			wantCode:       apitypes.ErrorCodeInvalidSignature,
		},
		"replayed update": {
			setup: []*apitypesv1.SetManifestRequest{
				{Manifest: first, Signature: sign(ownerKey, &history.Transition{ManifestHash: history.Digest(first)})},
				{Manifest: second, Signature: sign(ownerKey, &history.Transition{ManifestHash: history.Digest(second), PreviousTransitionHash: firstTransition})},
			},
			req: &apitypesv1.SetManifestRequest{
				Nonce:     nonce,
				Manifest:  second,
				Signature: sign(ownerKey, &history.Transition{ManifestHash: history.Digest(second), PreviousTransitionHash: firstTransition}),
			},
			wantHTTPStatus: http.StatusForbidden,
			wantCode:       apitypes.ErrorCodeInvalidSignature,
		},
		"stale previous transition hash": {
			setup: []*apitypesv1.SetManifestRequest{
				{Manifest: first, Signature: sign(ownerKey, &history.Transition{ManifestHash: history.Digest(first)})},
			},
			req: &apitypesv1.SetManifestRequest{
				Nonce:                  nonce,
				Manifest:               second,
				PreviousTransitionHash: make([]byte, history.HashSize),
				Signature:              sign(ownerKey, &history.Transition{ManifestHash: history.Digest(second), PreviousTransitionHash: firstTransition}),
			},
			wantHTTPStatus: http.StatusPreconditionFailed,
			wantCode:       apitypes.ErrorCodeManifestConflict,
		},
		"malformed manifest": {
			req:            &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: []byte(`{"Policies": 1}`)},
			wantHTTPStatus: http.StatusBadRequest,
			wantCode:       apitypes.ErrorCodeInvalidArgument,
		},
		"invalid nonce length": {
			req:            &apitypesv1.SetManifestRequest{Nonce: []byte{1, 2, 3}, Manifest: first},
			wantHTTPStatus: http.StatusBadRequest,
			wantCode:       apitypes.ErrorCodeInvalidArgument,
		},
		"invalid previous transition hash length": {
			req:            &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: first, PreviousTransitionHash: []byte{1}},
			wantHTTPStatus: http.StatusBadRequest,
			wantCode:       apitypes.ErrorCodeInvalidArgument,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			guard := newTestStateGuard()
			issuer := &stubIssuer{oid: asn1.ObjectIdentifier{1, 2, 3}}
			srv := httptest.NewServer(NewMux(issuer, guard, userapiserver.New(slog.Default(), guard, nil)))
			t.Cleanup(srv.Close)

			for _, setup := range tc.setup {
				setup.Nonce = nonce
				res := postSetManifest(t, srv, setup)
				require.Equal(http.StatusOK, res.StatusCode)
				res.Body.Close()
			}

			res := postSetManifest(t, srv, tc.req)
			defer res.Body.Close()
			require.Equal(tc.wantHTTPStatus, res.StatusCode)

			if tc.wantHTTPStatus != http.StatusOK {
				var apiErr apitypes.APIError
				require.NoError(json.NewDecoder(res.Body).Decode(&apiErr))
				require.Equal(tc.wantCode, apiErr.Code)
				require.Equal(tc.wantHTTPStatus, apiErr.StatusCode)
				if tc.wantErrNotContains != "" {
					require.NotContains(apiErr.Err, tc.wantErrNotContains)
				}
				return
			}

			var resp apitypesv1.SetManifestResponse
			require.NoError(json.NewDecoder(res.Body).Decode(&resp))
			require.Equal(constants.Version, resp.Version)
			require.Equal(issuer.oid, resp.AttestationType)
			require.NotEmpty(resp.RawAttestationDoc)
			require.Equal(tc.wantManifests, resp.Manifests)
			require.NotEmpty(resp.RootCA)
			require.NotEmpty(resp.MeshCA)

			if tc.wantSeedSharesDoc {
				// The document must be readable by `contrast recover`.
				var doc userapi.SeedShareDocument
				require.NoError(json.Unmarshal(resp.SeedSharesDoc, &doc))
				require.NotEmpty(doc.Salt)
			} else {
				require.Empty(resp.SeedSharesDoc)
			}

			// The transition clients compute from the manifest history is the one the report data covers.
			transitions := history.BuildTransitionChain(resp.Manifests)
			require.Equal(tc.wantTransition, transitions[len(transitions)-1].Digest())
			wantReportData, err := apitypesv1.ConstructSetManifestReportData(tc.req.Nonce, tc.wantTransition[:], &resp)
			require.NoError(err)
			require.Equal(wantReportData, issuer.gotReportData)
		})
	}
}

// TestSetManifestHandlerErrors checks the handling of failures that the actual UserAPI can't be made to produce easily.
func TestSetManifestHandlerErrors(t *testing.T) {
	validRequest := &apitypesv1.SetManifestRequest{Nonce: nonce, Manifest: []byte("{}")}

	for name, tc := range map[string]struct {
		setter      *stubManifestSetter
		issuer      *stubIssuer
		guard       *stubGuard
		method      string
		contentType string

		wantHTTPStatus int
		wantCode       apitypes.ErrorCode
	}{
		"wrong HTTP method": {
			method:         http.MethodGet,
			wantHTTPStatus: http.StatusMethodNotAllowed,
			wantCode:       apitypes.ErrorCodeInvalidArgument,
		},
		"wrong Content-Type": {
			contentType:    "text/plain",
			wantHTTPStatus: http.StatusUnsupportedMediaType,
			wantCode:       apitypes.ErrorCodeInvalidArgument,
		},
		"needs recovery": {
			setter:         &stubManifestSetter{err: status.Error(codes.FailedPrecondition, userapiserver.ErrNeedsRecovery.Error())},
			wantHTTPStatus: http.StatusPreconditionFailed,
		},
		"permission denied": {
			setter:         &stubManifestSetter{err: status.Error(codes.PermissionDenied, "changes to seedshare owners are not allowed")},
			wantHTTPStatus: http.StatusForbidden,
			wantCode:       apitypes.ErrorCodePermissionDenied,
		},
		"internal error": {
			setter:         &stubManifestSetter{err: status.Error(codes.Internal, "boom")},
			wantHTTPStatus: http.StatusInternalServerError,
			wantCode:       apitypes.ErrorCodeInternal,
		},
		"history unavailable": {
			guard:          &stubGuard{getHistoryErr: assert.AnError},
			wantHTTPStatus: http.StatusInternalServerError,
			wantCode:       apitypes.ErrorCodeInternal,
		},
		"unable to get attestation": {
			issuer:         &stubIssuer{issueErr: assert.AnError},
			wantHTTPStatus: http.StatusInternalServerError,
			wantCode:       apitypes.ErrorCodeInternal,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			caInstance := newTestCA(t)
			if tc.guard == nil {
				tc.guard = &stubGuard{}
			}
			tc.guard.ca = caInstance
			if tc.setter == nil {
				state, err := tc.guard.GetState(t.Context())
				require.NoError(err)
				tc.setter = &stubManifestSetter{state: state}
			}
			if tc.issuer == nil {
				tc.issuer = &stubIssuer{oid: asn1.ObjectIdentifier{1, 2, 3}}
			}
			handler := &SetManifestHandler{Issuer: tc.issuer, StateGuard: tc.guard, UserAPI: tc.setter}

			body, err := json.Marshal(validRequest)
			require.NoError(err)
			method := apitypesv1.SetManifestMethod
			if tc.method != "" {
				method = tc.method
			}
			req := httptest.NewRequestWithContext(t.Context(), method, apitypesv1.ManifestPath, bytes.NewReader(body))
			contentType := "application/json"
			if tc.contentType != "" {
				contentType = tc.contentType
			}
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			require.Equal(tc.wantHTTPStatus, res.StatusCode)
			var apiErr apitypes.APIError
			require.NoError(json.NewDecoder(res.Body).Decode(&apiErr))
			if tc.wantCode != "" {
				require.Equal(tc.wantCode, apiErr.Code)
			}
		})
	}
}

func postSetManifest(t *testing.T, srv *httptest.Server, req *apitypesv1.SetManifestRequest) *http.Response {
	t.Helper()
	body, err := json.Marshal(req)
	require.NoError(t, err)
	httpReq, err := http.NewRequestWithContext(t.Context(), apitypesv1.SetManifestMethod, srv.URL+apitypesv1.ManifestPath, bytes.NewReader(body))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := srv.Client().Do(httpReq)
	require.NoError(t, err)
	return res
}

func newTestStateGuard() *stateguard.Guard {
	store := aferostore.New(&afero.Afero{Fs: afero.NewMemMapFs()})
	hist := history.NewWithStore(slog.Default(), store)
	return stateguard.New(hist, prometheus.NewRegistry(), slog.Default())
}

type stubManifestSetter struct {
	state *stateguard.State
	err   error
}

func (s *stubManifestSetter) SetManifestSigned(context.Context, *userapi.SetManifestRequest) (*userapi.SetManifestResponse, *stateguard.State, error) {
	if s.err != nil {
		return nil, nil, s.err
	}
	return &userapi.SetManifestResponse{}, s.state, nil
}
