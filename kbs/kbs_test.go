package kbs

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.step.sm/crypto/jose"
)

func TestAuthHandler(t *testing.T) {
	testCases := map[string]struct {
		server     server
		req        testRequest
		wantStatus int
	}{
		"bad request": {
			server: server{
				teePubKeys:    make(map[string]*jose.JSONWebKey),
				cookieToNonce: make(map[string]string),
			},
			req:        testRequest{method: http.MethodPost, body: ""},
			wantStatus: http.StatusBadRequest,
		},
		"empty request": {
			server: server{
				teePubKeys:    make(map[string]*jose.JSONWebKey),
				cookieToNonce: make(map[string]string),
			},
			req:        testRequest{method: http.MethodPost, body: "{}"},
			wantStatus: http.StatusOK,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			server := httptest.NewServer(tc.server.newHandler())
			defer server.Close()
			client := server.Client()

			req, err := http.NewRequest(tc.req.method, server.URL+"/kbs/v0/auth", bytes.NewBufferString(tc.req.body))
			require.NoError(err)
			resp, err := client.Do(req)
			require.NoError(err)

			assert.Equal(tc.wantStatus, resp.StatusCode)
			if tc.wantStatus != http.StatusOK {
				assert.Len(tc.server.teePubKeys, 0)
				assert.Len(tc.server.cookieToNonce, 0)
				return
			}
			assert.Len(tc.server.teePubKeys, 0)
			assert.Len(tc.server.cookieToNonce, 1)
			body, err := io.ReadAll(resp.Body)
			assert.NoError(err)
			var challengeResp Challenge
			err = json.Unmarshal(body, &challengeResp)
			assert.NoError(err)
			assert.NotEmpty(challengeResp.Nonce)
			cookies := resp.Cookies()
			var sessionCookie *http.Cookie
			for _, cookie := range cookies {
				if cookie.Name == sessionIDCookieName {
					sessionCookie = cookie
					break
				}
			}
			assert.NotNil(sessionCookie)
			assert.NotEmpty(sessionCookie.Value)
		})
	}
}

func TestAttestHandler(t *testing.T) {
	testCases := map[string]struct {
		server     server
		req        testRequest
		wantStatus int
	}{
		"successfull attestation": {
			server: server{
				teePubKeys:    make(map[string]*jose.JSONWebKey),
				cookieToNonce: map[string]string{sessionIDCookieName: "someNonce"},
			},
			req: testRequest{
				method: http.MethodPost,
				body:   `{"teePubKey":{"kty":"EC","crv":"P-256","x":"x","y":"y"}}`,
				cookie: &http.Cookie{Name: sessionIDCookieName, Value: sessionIDCookieName},
			},
			wantStatus: http.StatusOK,
		},
		// "no cookie": {
		// 	server: server{
		// 		teePubKeys:    make(map[string]*jose.JSONWebKey),
		// 		cookieToNonce: make(map[string]string),
		// 	},
		// 	req:        testRequest{method: http.MethodPost, body: ""},
		// 	wantStatus: http.StatusBadRequest,
		// },
		// "unknown cookie": {
		// 	server: server{
		// 		teePubKeys:    make(map[string]*jose.JSONWebKey),
		// 		cookieToNonce: make(map[string]string),
		// 	},
		// 	req: testRequest{
		// 		method: http.MethodPost,
		// 		body:   "",
		// 		cookie: http.Cookie{Name: sessionIDCookieName, Value: "someValue"},
		// 	},
		// 	wantStatus: http.StatusBadRequest,
		// },
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			server := httptest.NewServer(tc.server.newHandler())
			defer server.Close()
			client := server.Client()

			req, err := http.NewRequest(tc.req.method, server.URL+"/kbs/v0/attest", bytes.NewBufferString(tc.req.body))
			require.NoError(err)
			if tc.req.cookie != nil {
				req.AddCookie(tc.req.cookie)
			}
			resp, err := client.Do(req)
			require.NoError(err)

			assert.Equal(tc.wantStatus, resp.StatusCode)
			assert.Len(tc.server.teePubKeys, 0)
			assert.Len(tc.server.cookieToNonce, 0)
		})
	}
}

type testRequest struct {
	method string
	body   string
	cookie *http.Cookie
}
