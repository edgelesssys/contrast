// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/hpke"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/edgelesssys/contrast/coordinator/internal/userapi"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var nonce = make([]byte, 32)

func TestAttestationHandler(t *testing.T) {
	testCases := map[string]struct {
		request *apitypesv1.AttestationRequest
		method  string

		malformedBody   bool
		contentType     string
		skipContentType bool

		guard  *stubGuard
		issuer *stubIssuer

		expStatus int
		expErr    error
	}{
		"invalid nonce length": {
			request:   &apitypesv1.AttestationRequest{Nonce: []byte{1, 2, 3}},
			expStatus: http.StatusBadRequest,
			expErr:    errNonceLength,
		},
		"wrong HTTP method": {
			method:    http.MethodGet,
			expStatus: http.StatusMethodNotAllowed,
			expErr:    errMethodNotAllowed,
		},
		"no body": {
			request:   &apitypesv1.AttestationRequest{},
			expStatus: http.StatusBadRequest,
			expErr:    errNonceLength,
		},
		"empty body": {
			expStatus: http.StatusBadRequest,
			expErr:    errNonceLength,
		},
		"malformed body": {
			malformedBody: true,
			expStatus:     http.StatusBadRequest,
			expErr:        errNonceLength,
		},
		"no Content-Type": {
			request:         &apitypesv1.AttestationRequest{Nonce: nonce},
			skipContentType: true,
			expStatus:       http.StatusBadRequest,
		},
		"wrong Content-Type": {
			request:     &apitypesv1.AttestationRequest{Nonce: nonce},
			contentType: "text/html",
			expStatus:   http.StatusUnsupportedMediaType,
			expErr:      errContentType,
		},
		"no state": {
			request:   &apitypesv1.AttestationRequest{Nonce: nonce},
			guard:     &stubGuard{getStateErr: stateguard.ErrNoState},
			expStatus: http.StatusPreconditionFailed,
			expErr:    userapi.ErrNoManifest,
		},
		"stale state": {
			request:   &apitypesv1.AttestationRequest{Nonce: nonce},
			guard:     &stubGuard{getStateErr: stateguard.ErrStaleState},
			expStatus: http.StatusPreconditionFailed,
			expErr:    userapi.ErrNeedsRecovery,
		},
		"unknown error during GetState": {
			request:   &apitypesv1.AttestationRequest{Nonce: nonce},
			guard:     &stubGuard{getStateErr: assert.AnError},
			expStatus: http.StatusInternalServerError,
			expErr:    errGettingState,
		},
		"unknown error during GetHistory": {
			request:   &apitypesv1.AttestationRequest{Nonce: nonce},
			guard:     &stubGuard{getHistoryErr: assert.AnError},
			expStatus: http.StatusInternalServerError,
			expErr:    errGettingHistory,
		},
		"unable to get attestation": {
			request:   &apitypesv1.AttestationRequest{Nonce: nonce},
			issuer:    &stubIssuer{issueErr: assert.AnError},
			expStatus: http.StatusInternalServerError,
			expErr:    errGettingAttestation,
		},
		"success": {
			request:   &apitypesv1.AttestationRequest{Nonce: nonce},
			expStatus: http.StatusOK,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			if tc.guard == nil {
				tc.guard = &stubGuard{}
			}
			meshKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[1])
			rootKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[2])
			ca, err := ca.New(rootKey, meshKey)
			require.NoError(err)
			tc.guard.ca = ca

			expectedOID := asn1.ObjectIdentifier{1, 2, 3}
			if tc.issuer == nil {
				tc.issuer = &stubIssuer{oid: expectedOID}
			}

			handler := &AttestationHandler{
				StateGuard: tc.guard,
				Issuer:     tc.issuer,
			}

			bodyBytes, err := json.Marshal(tc.request)
			require.NoError(err)

			if tc.malformedBody {
				// Marshalling something unexpected
				nonesenseBytes, err := json.Marshal(handler)
				require.NoError(err)
				bodyBytes = nonesenseBytes
			}

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/attest", bytes.NewReader(bodyBytes))
			contentType := "application/json"
			if tc.contentType != "" {
				contentType = tc.contentType
			}
			if !tc.skipContentType {
				req.Header.Set("Content-Type", contentType)
			}
			if tc.method != "" {
				req.Method = tc.method
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			require.Equal(tc.expStatus, res.StatusCode)

			if tc.expErr != nil {
				var apiErr apitypesv1.AttestationError
				require.NoError(json.NewDecoder(res.Body).Decode(&apiErr))
				require.Contains(apiErr.Err, tc.expErr.Error())
				require.Equal(tc.expStatus, apiErr.StatusCode)
			} else if res.StatusCode == http.StatusOK {
				var resp apitypesv1.AttestationResponse
				require.NoError(json.NewDecoder(res.Body).Decode(&resp))
				require.Equal(constants.Version, resp.Version)
				require.Equal(expectedOID, resp.AttestationType)
				require.NotEmpty(resp.RawAttestationDoc)
			}
		})
	}
}

// TestLegacyAttestationResponse ensures the unversioned endpoint keeps its released response,
// without the fields that were added for the versioned endpoint.
func TestLegacyAttestationResponse(t *testing.T) {
	require := require.New(t)

	handler := &AttestationHandler{
		StateGuard: &stubGuard{ca: newTestCA(t)},
		Issuer:     &stubIssuer{oid: asn1.ObjectIdentifier{1, 2, 3}},
	}
	resp, status, err := handler.getResponse(t.Context(), nonce)
	require.NoError(err)
	require.Equal(http.StatusOK, status)

	respJSON, err := json.Marshal(resp)
	require.NoError(err)
	var fields map[string]json.RawMessage
	require.NoError(json.Unmarshal(respJSON, &fields))
	require.ElementsMatch(
		[]string{"version", "raw_attestation_doc", "attestation_type", "manifests", "policies", "root_ca", "mesh_ca"},
		slices.Collect(maps.Keys(fields)),
	)
}

func TestVersionedAttestationHandler(t *testing.T) {
	capabilitiesDigest := NewCapabilitiesHandler().Digest()
	// stubGuard creates its state from no manifest bytes, see [stateguard.NewStateForTest].
	transition := (&history.Transition{ManifestHash: history.Digest(nil)}).Digest()
	transitionDigest := transition[:]

	testCases := map[string]struct {
		guard  *stubGuard
		issuer *stubIssuer

		wantStatus           apitypesv1.Status
		wantTransitionDigest []byte
		wantHTTPStatus       int
		wantErr              error
	}{
		"active": {
			guard:                &stubGuard{},
			wantStatus:           apitypesv1.StatusActive,
			wantTransitionDigest: transitionDigest,
		},
		"uninitialized": {
			guard:      &stubGuard{getStateErr: stateguard.ErrNoState},
			wantStatus: apitypesv1.StatusUninitialized,
		},
		"stale": {
			guard:      &stubGuard{getStateErr: stateguard.ErrStaleState},
			wantStatus: apitypesv1.StatusStale,
		},
		"unknown error during GetState": {
			guard:          &stubGuard{getStateErr: assert.AnError},
			wantHTTPStatus: http.StatusInternalServerError,
			wantErr:        errGettingState,
		},
		"unknown error during GetHistory": {
			guard:          &stubGuard{getHistoryErr: assert.AnError},
			wantHTTPStatus: http.StatusInternalServerError,
			wantErr:        errGettingHistory,
		},
		"unable to get attestation": {
			guard:          &stubGuard{},
			issuer:         &stubIssuer{issueErr: assert.AnError},
			wantHTTPStatus: http.StatusInternalServerError,
			wantErr:        errGettingAttestation,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			tc.guard.ca = newTestCA(t)
			expectedOID := asn1.ObjectIdentifier{1, 2, 3}
			if tc.issuer == nil {
				tc.issuer = &stubIssuer{oid: expectedOID}
			}
			pendingKeys := NewPendingKeys()
			handler := &VersionedAttestationHandler{
				StateGuard:         tc.guard,
				Issuer:             tc.issuer,
				CapabilitiesDigest: capabilitiesDigest,
				PendingKeys:        pendingKeys,
			}

			body, err := json.Marshal(&apitypesv1.AttestationRequest{Nonce: nonce})
			require.NoError(err)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, apitypesv1.AttestPath, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			if tc.wantErr != nil {
				require.Equal(tc.wantHTTPStatus, res.StatusCode)
				var apiErr apitypesv1.AttestationError
				require.NoError(json.NewDecoder(res.Body).Decode(&apiErr))
				require.Contains(apiErr.Err, tc.wantErr.Error())
				_, ok := pendingKeys.Take([32]byte(nonce))
				require.False(ok, "no key must be kept for a failed exchange")
				return
			}
			require.Equal(http.StatusOK, res.StatusCode)

			var resp apitypesv1.AttestationResponse
			require.NoError(json.NewDecoder(res.Body).Decode(&resp))
			require.Equal(constants.Version, resp.Version)
			require.Equal(expectedOID, resp.AttestationType)
			require.NotEmpty(resp.RawAttestationDoc)
			require.Equal(tc.wantStatus, resp.Status)
			require.Equal(capabilitiesDigest[:], resp.CapabilitiesDigest)
			if tc.wantStatus == apitypesv1.StatusActive {
				require.NotEmpty(resp.RootCA)
				require.NotEmpty(resp.MeshCA)
			} else {
				require.Equal(apitypesv1.CoordinatorState{}, resp.CoordinatorState)
			}

			// A client must be able to reproduce the report data from the response alone.
			wantReportData, err := apitypesv1.ConstructAttestationReportData(nonce, tc.wantTransitionDigest, &resp)
			require.NoError(err)
			require.Equal(wantReportData, tc.issuer.gotReportData)

			// The private key of the attested public key is kept for exactly one follow-up request.
			hpkeKey, ok := pendingKeys.Take([32]byte(nonce))
			require.True(ok)
			require.Equal(hpke.MLKEM768X25519().ID(), hpkeKey.KEM().ID())
			require.Equal(resp.HPKEPubKey, hpkeKey.PublicKey().Bytes())
			_, ok = pendingKeys.Take([32]byte(nonce))
			require.False(ok)

			// The key is usable with the primitives RFC 015 selects.
			pub, err := hpke.MLKEM768X25519().NewPublicKey(resp.HPKEPubKey)
			require.NoError(err)
			ciphertext, err := hpke.Seal(pub, hpke.HKDFSHA256(), hpke.AES256GCM(), []byte("info"), []byte("secret"))
			require.NoError(err)
			plaintext, err := hpke.Open(hpkeKey, hpke.HKDFSHA256(), hpke.AES256GCM(), []byte("info"), ciphertext)
			require.NoError(err)
			require.Equal([]byte("secret"), plaintext)
		})
	}
}

// TestVersionedAttestationFreshKeys ensures every exchange gets a key of its own.
func TestVersionedAttestationFreshKeys(t *testing.T) {
	require := require.New(t)

	handler := &VersionedAttestationHandler{
		StateGuard:  &stubGuard{ca: newTestCA(t)},
		Issuer:      &stubIssuer{oid: asn1.ObjectIdentifier{1, 2, 3}},
		PendingKeys: NewPendingKeys(),
	}
	first, _, err := handler.getResponse(t.Context(), nonce)
	require.NoError(err)
	second, _, err := handler.getResponse(t.Context(), nonce)
	require.NoError(err)
	require.NotEqual(first.HPKEPubKey, second.HPKEPubKey)

	// The key of the later exchange replaces the one of the earlier exchange with the same nonce.
	hpkeKey, ok := handler.PendingKeys.Take([32]byte(nonce))
	require.True(ok)
	require.Equal(second.HPKEPubKey, hpkeKey.PublicKey().Bytes())
}

func newTestCA(t *testing.T) *ca.CA {
	t.Helper()
	meshKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[1])
	rootKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[2])
	ca, err := ca.New(rootKey, meshKey)
	require.NoError(t, err)
	return ca
}

type stubIssuer struct {
	oid           asn1.ObjectIdentifier
	issueErr      error
	gotReportData [64]byte
	atls.Issuer
}

func (s *stubIssuer) OID() asn1.ObjectIdentifier {
	return s.oid
}

func (s *stubIssuer) Issue(_ context.Context, reportData [64]byte) (quote []byte, err error) {
	if s.issueErr != nil {
		return nil, s.issueErr
	}
	s.gotReportData = reportData
	return []byte("fake-attestation"), nil
}

type stubGuard struct {
	ca                *ca.CA
	minimumAPIVersion string
	getStateErr       error
	getHistoryErr     error
	stateguard.Guard
}

func (s *stubGuard) GetState(context.Context) (*stateguard.State, error) {
	if s.getStateErr != nil {
		return nil, s.getStateErr
	}
	m := &manifest.Manifest{MinimumAPIVersion: s.minimumAPIVersion}
	policyHash := sha256.Sum256(nil)
	policyHashHex := manifest.NewHexString(policyHash[:])
	m.Policies = map[manifest.HexString]manifest.PolicyEntry{
		policyHashHex: {
			SANs:             []string{"test"},
			WorkloadSecretID: "test",
		},
	}

	return stateguard.NewStateForTest(nil, m, nil, s.ca), nil
}

func (s *stubGuard) GetHistory(context.Context) ([][]byte, map[manifest.HexString][]byte, error) {
	if s.getHistoryErr != nil {
		return nil, nil, s.getHistoryErr
	}
	return nil, nil, nil
}
