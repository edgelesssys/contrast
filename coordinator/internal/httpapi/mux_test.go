// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/asn1"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/stretchr/testify/require"
)

func TestMuxAttestation(t *testing.T) {
	meshKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[1])
	rootKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[2])
	ca, err := ca.New(rootKey, meshKey)
	require.NoError(t, err)

	issuer := &stubIssuer{oid: asn1.ObjectIdentifier{1, 2, 3}}
	srv := httptest.NewServer(NewMux(issuer, &stubGuard{ca: ca}))
	t.Cleanup(srv.Close)

	do := func(t *testing.T, method, path string, body []byte) []byte {
		req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		res, err := srv.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
		respBody, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		return respBody
	}

	attestRequest, err := json.Marshal(apitypesv1.AttestationRequest{Nonce: nonce})
	require.NoError(t, err)
	transitionDigest := make([]byte, history.HashSize)

	t.Run("legacy", func(t *testing.T) {
		require := require.New(t)

		var resp apitypesv1.AttestationResponse
		require.NoError(json.Unmarshal(do(t, http.MethodPost, apitypesv1.LegacyAttestPath, attestRequest), &resp))
		require.Equal(apitypesv1.ConstructReportData(nonce, transitionDigest, &resp.CoordinatorState), issuer.gotReportData)
	})
}
