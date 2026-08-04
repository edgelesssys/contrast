// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	coorduserapi "github.com/edgelesssys/contrast/coordinator/internal/userapi"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSetManifestHandler(t *testing.T) {
	validRequest := &apitypes.SetManifestRequest{
		Manifest:  []byte(`{"foo":"bar"}`),
		Policies:  [][]byte{[]byte("policy1")},
		Signature: []byte("signature"),
	}

	testCases := map[string]struct {
		request *apitypes.SetManifestRequest
		method  string

		malformedBody   bool
		contentType     string
		skipContentType bool

		setter *stubManifestSetter

		expStatus int
		expErr    string
	}{
		"success": {
			request:   validRequest,
			expStatus: http.StatusOK,
		},
		"wrong HTTP method": {
			method:    http.MethodGet,
			expStatus: http.StatusMethodNotAllowed,
		},
		"no Content-Type": {
			request:         validRequest,
			skipContentType: true,
			expStatus:       http.StatusBadRequest,
		},
		"wrong Content-Type": {
			request:     validRequest,
			contentType: "text/html",
			expStatus:   http.StatusUnsupportedMediaType,
			expErr:      errContentType.Error(),
		},
		"malformed body": {
			malformedBody: true,
			expStatus:     http.StatusBadRequest,
		},
		"invalid manifest": {
			request:   validRequest,
			setter:    &stubManifestSetter{err: status.Error(codes.InvalidArgument, "unmarshaling manifest")},
			expStatus: http.StatusBadRequest,
			expErr:    "unmarshaling manifest",
		},
		"unauthorized": {
			request:   validRequest,
			setter:    &stubManifestSetter{err: status.Error(codes.PermissionDenied, "validating manifest signature")},
			expStatus: http.StatusForbidden,
			expErr:    "validating manifest signature",
		},
		"needs recovery": {
			request:   validRequest,
			setter:    &stubManifestSetter{err: status.Error(codes.FailedPrecondition, coorduserapi.ErrNeedsRecovery.Error())},
			expStatus: http.StatusPreconditionFailed,
			expErr:    coorduserapi.ErrNeedsRecovery.Error(),
		},
		"internal error": {
			request:   validRequest,
			setter:    &stubManifestSetter{err: status.Error(codes.Internal, "getting state")},
			expStatus: http.StatusInternalServerError,
			expErr:    "getting state",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			if tc.setter == nil {
				tc.setter = &stubManifestSetter{}
			}
			handler := &SetManifestHandler{UserAPI: tc.setter}

			bodyBytes, err := json.Marshal(tc.request)
			require.NoError(err)
			if tc.malformedBody {
				bodyBytes = []byte("not json")
			}

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/manifest", bytes.NewReader(bodyBytes))
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

			switch {
			case tc.expStatus == http.StatusMethodNotAllowed:
				// No body expected.
			case tc.expStatus != http.StatusOK:
				var apiErr apitypes.APIError
				require.NoError(json.NewDecoder(res.Body).Decode(&apiErr))
				assert.Equal(constants.Version, apiErr.Version)
				assert.Equal(tc.expStatus, apiErr.StatusCode)
				if tc.expErr != "" {
					assert.Contains(apiErr.Err, tc.expErr)
				}
			default:
				// The request must have been passed through unchanged.
				assert.Equal(tc.request.Manifest, tc.setter.gotReq.GetManifest())
				assert.Equal(tc.request.Policies, tc.setter.gotReq.GetPolicies())
				assert.Equal(tc.request.Signature, tc.setter.gotReq.GetSignature())

				var resp apitypes.SetManifestResponse
				require.NoError(json.NewDecoder(res.Body).Decode(&resp))
				assert.Equal(constants.Version, resp.Version)
				assert.Equal([]byte("root-ca"), resp.RootCA)
				assert.Equal([]byte("mesh-ca"), resp.MeshCA)
				require.NotNil(resp.SeedSharesDoc)
				assert.Equal([]byte("salt"), resp.SeedSharesDoc.Salt)
				require.Len(resp.SeedSharesDoc.SeedShares, 1)
				assert.Equal("public-key", resp.SeedSharesDoc.SeedShares[0].PublicKey)
				assert.Equal([]byte("encrypted-seed"), resp.SeedSharesDoc.SeedShares[0].EncryptedSeed)
			}
		})
	}
}

// TestSetManifestHandlerNoSeedShares ensures the optional seed share document stays unset
// for updates to an existing manifest.
func TestSetManifestHandlerNoSeedShares(t *testing.T) {
	require := require.New(t)

	setter := &stubManifestSetter{resp: &userapi.SetManifestResponse{
		RootCA: []byte("root-ca"),
		MeshCA: []byte("mesh-ca"),
	}}
	handler := &SetManifestHandler{UserAPI: setter}

	bodyBytes, err := json.Marshal(&apitypes.SetManifestRequest{Manifest: []byte("{}")})
	require.NoError(err)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/manifest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()

	require.Equal(http.StatusOK, res.StatusCode)
	var resp apitypes.SetManifestResponse
	require.NoError(json.NewDecoder(res.Body).Decode(&resp))
	require.Nil(resp.SeedSharesDoc)
}

type stubManifestSetter struct {
	resp   *userapi.SetManifestResponse
	err    error
	gotReq *userapi.SetManifestRequest
}

func (s *stubManifestSetter) SetManifest(_ context.Context, req *userapi.SetManifestRequest) (*userapi.SetManifestResponse, error) {
	s.gotReq = req
	if s.err != nil {
		return nil, s.err
	}
	if s.resp != nil {
		return s.resp, nil
	}
	return &userapi.SetManifestResponse{
		RootCA: []byte("root-ca"),
		MeshCA: []byte("mesh-ca"),
		SeedSharesDoc: &userapi.SeedShareDocument{
			Salt: []byte("salt"),
			SeedShares: []*userapi.SeedShare{{
				PublicKey:     "public-key",
				EncryptedSeed: []byte("encrypted-seed"),
			}},
		},
	}, nil
}
