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

	"github.com/edgelesssys/contrast/apitypes"
	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/stretchr/testify/require"
)

// TestMuxAttestation ensures the endpoints are wired up so that clients can reproduce the report data:
// the legacy endpoint keeps its released response and report data layout, and the versioned endpoint binds
// the capabilities the Coordinator actually serves.
func TestMuxAttestation(t *testing.T) {
	meshKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[1])
	rootKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[2])
	ca, err := ca.New(rootKey, meshKey)
	require.NoError(t, err)

	issuer := &stubIssuer{oid: asn1.ObjectIdentifier{1, 2, 3}}
	guard := &stubGuard{ca: ca}
	srv := httptest.NewServer(NewMux(issuer, guard, &stubManifestSetter{}))
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
	state, err := guard.GetState(t.Context())
	require.NoError(t, err)
	transitionDigest := state.LatestTransition().TransitionHash[:]

	t.Run("legacy", func(t *testing.T) {
		require := require.New(t)

		var resp apitypesv1.AttestationResponse
		require.NoError(json.Unmarshal(do(t, apitypesv1.AttestMethod, apitypesv1.LegacyAttestPath, attestRequest), &resp))
		require.Nil(resp.CapabilitiesDigest)
		require.Empty(resp.Status)
		require.Nil(resp.HPKEPubKey)
		require.Equal(apitypesv1.ConstructReportData(nonce, transitionDigest, &resp.CoordinatorState), issuer.gotReportData)
	})

	t.Run("v1", func(t *testing.T) {
		require := require.New(t)

		var capabilities apitypes.CapabilitiesResponse
		require.NoError(json.Unmarshal(do(t, apitypes.CapabilitiesMethod, apitypes.CapabilitiesPath, nil), &capabilities))
		require.Contains(capabilities.APIVersions, apitypesv1.Version)
		capabilitiesDigest := capabilities.Digest()

		var resp apitypesv1.AttestationResponse
		require.NoError(json.Unmarshal(do(t, apitypesv1.AttestMethod, apitypesv1.AttestPath, attestRequest), &resp))
		require.Equal(capabilitiesDigest[:], resp.CapabilitiesDigest)
		require.Equal(apitypesv1.StatusActive, resp.Status)
		require.NotEmpty(resp.HPKEPubKey)

		wantReportData, err := apitypesv1.ConstructAttestationReportData(nonce, transitionDigest, &resp)
		require.NoError(err)
		require.Equal(wantReportData, issuer.gotReportData)
	})
}
