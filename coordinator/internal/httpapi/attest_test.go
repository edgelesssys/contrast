// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/edgelesssys/contrast/coordinator/internal/userapi"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/httpapi"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var nonce = make([]byte, 32)

func TestAttestationHandler(t *testing.T) {
	testCases := map[string]struct {
		request *httpapi.AttestationRequest
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
			request:   &httpapi.AttestationRequest{Nonce: []byte{1, 2, 3}},
			expStatus: http.StatusBadRequest,
			expErr:    errNonceLength,
		},
		"wrong HTTP method": {
			method:    http.MethodGet,
			expStatus: http.StatusMethodNotAllowed,
		},
		"no body": {
			request:   &httpapi.AttestationRequest{},
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
			request:         &httpapi.AttestationRequest{Nonce: nonce},
			skipContentType: true,
			expStatus:       http.StatusBadRequest,
		},
		"wrong Content-Type": {
			request:     &httpapi.AttestationRequest{Nonce: nonce},
			contentType: "text/html",
			expStatus:   http.StatusUnsupportedMediaType,
			expErr:      errContentType,
		},
		"no state": {
			request:   &httpapi.AttestationRequest{Nonce: nonce},
			guard:     &stubGuard{getStateErr: stateguard.ErrNoState},
			expStatus: http.StatusPreconditionFailed,
			expErr:    userapi.ErrNoManifest,
		},
		"stale state": {
			request:   &httpapi.AttestationRequest{Nonce: nonce},
			guard:     &stubGuard{getStateErr: stateguard.ErrStaleState},
			expStatus: http.StatusPreconditionFailed,
			expErr:    userapi.ErrNeedsRecovery,
		},
		"unknown error during GetState": {
			request:   &httpapi.AttestationRequest{Nonce: nonce},
			guard:     &stubGuard{getStateErr: assert.AnError},
			expStatus: http.StatusInternalServerError,
			expErr:    errGettingState,
		},
		"unknown error during GetHistory": {
			request:   &httpapi.AttestationRequest{Nonce: nonce},
			guard:     &stubGuard{getHistoryErr: assert.AnError},
			expStatus: http.StatusInternalServerError,
			expErr:    errGettingHistory,
		},
		"unable to get attestation": {
			request:   &httpapi.AttestationRequest{Nonce: nonce},
			issuer:    &stubIssuer{issueErr: assert.AnError},
			expStatus: http.StatusInternalServerError,
			expErr:    errGettingAttestation,
		},
		"success": {
			request:   &httpapi.AttestationRequest{Nonce: nonce},
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

			if tc.issuer == nil {
				tc.issuer = &stubIssuer{}
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

			req := httptest.NewRequest(http.MethodPost, "/attest", bytes.NewReader(bodyBytes))
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
				var apiErr httpapi.AttestationError
				require.NoError(json.NewDecoder(res.Body).Decode(&apiErr))
				require.Contains(apiErr.Err, tc.expErr.Error())
			} else if res.StatusCode == http.StatusOK {
				var resp httpapi.AttestationResponse
				require.NoError(json.NewDecoder(res.Body).Decode(&resp))
				require.Equal(constants.Version, resp.Version)
				require.NotEmpty(resp.RawAttestationDoc)
			}
		})
	}
}

type stubIssuer struct {
	issueErr error
	atls.Issuer
}

func (s *stubIssuer) Issue(_ context.Context, _ [64]byte) (quote []byte, err error) {
	if s.issueErr != nil {
		return nil, s.issueErr
	}
	return []byte("fake-attestation"), nil
}

type stubGuard struct {
	ca            *ca.CA
	getStateErr   error
	getHistoryErr error
	stateguard.Guard
}

func (s *stubGuard) GetState(context.Context) (*stateguard.State, error) {
	if s.getStateErr != nil {
		return nil, s.getStateErr
	}
	m := &manifest.Manifest{}
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
