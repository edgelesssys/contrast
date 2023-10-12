package kbs

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/katexochen/coordinator-kbs/ca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.step.sm/crypto/jose"
)

func TestResourceHandler(t *testing.T) {
	testCases := map[string]struct {
		server     *server
		reqCookie  *http.Cookie
		wantStatus int
	}{
		"valid request": {
			server: &server{
				cookieToTEEPubKey: map[string]*jose.JSONWebKey{"foo": genJWK()},
				cookieToNonce:     make(map[string]string),
				certGen:           mustVal(ca.NewCA()),
			},
			reqCookie:  &http.Cookie{Name: sessionIDCookieName, Value: "foo"},
			wantStatus: http.StatusOK,
		},
		"no cookie": {
			server: &server{
				cookieToTEEPubKey: make(map[string]*jose.JSONWebKey),
				cookieToNonce:     make(map[string]string),
			},
			wantStatus: http.StatusBadRequest,
		},
		"unknown cookie": {
			server: &server{
				cookieToTEEPubKey: make(map[string]*jose.JSONWebKey),
				cookieToNonce:     make(map[string]string),
			},
			reqCookie:  &http.Cookie{Name: sessionIDCookieName, Value: "unknown"},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			server := httptest.NewServer(tc.server.newHandler())
			defer server.Close()
			client := server.Client()

			req, err := http.NewRequest(http.MethodGet, server.URL+"/kbs/v0/resource/foo/bar/batz", http.NoBody)
			require.NoError(err)
			if tc.reqCookie != nil {
				req.AddCookie(tc.reqCookie)
			}
			resp, err := client.Do(req)
			require.NoError(err)

			assert.Equal(tc.wantStatus, resp.StatusCode)
			if tc.wantStatus != http.StatusOK {
				return
			}
		})
	}
}

func TestAuthHandler(t *testing.T) {
	testCases := map[string]struct {
		server     server
		req        testRequest
		wantStatus int
	}{
		"bad request": {
			server: server{
				cookieToTEEPubKey: make(map[string]*jose.JSONWebKey),
				cookieToNonce:     make(map[string]string),
			},
			req:        testRequest{method: http.MethodPost, body: ""},
			wantStatus: http.StatusBadRequest,
		},
		"empty request": {
			server: server{
				cookieToTEEPubKey: make(map[string]*jose.JSONWebKey),
				cookieToNonce:     make(map[string]string),
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
				assert.Len(tc.server.cookieToTEEPubKey, 0)
				assert.Len(tc.server.cookieToNonce, 0)
				return
			}
			assert.Len(tc.server.cookieToTEEPubKey, 0)
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
				cookieToTEEPubKey: make(map[string]*jose.JSONWebKey),
				privKey:           genPrivKey(),
				cookieToNonce:     map[string]string{sessionIDCookieName: "someNonce"},
			},
			req: testRequest{
				method: http.MethodPost,
				body:   `{"tee-pubkey":{"use":"sig","kty":"EC","kid":"_pmROS94lb42QYA2ZWLt6sHITkInru_sJSkF8IqxonA","crv":"P-256","alg":"ES256","x":"mTwy33-eUlbI7uR1pG-SQzEaOHVvMB1aW0mTJKuRnXQ","y":"uK3u9tcDbSr4VoO4J1BaFh0ttZ6aEykbsiwEpvtw3BA"}}`,
				cookie: &http.Cookie{Name: sessionIDCookieName, Value: sessionIDCookieName},
			},
			wantStatus: http.StatusOK,
		},
		"no cookie": {
			server: server{
				cookieToTEEPubKey: make(map[string]*jose.JSONWebKey),
				cookieToNonce:     make(map[string]string),
			},
			req:        testRequest{method: http.MethodPost, body: ""},
			wantStatus: http.StatusBadRequest,
		},
		"unknown cookie": {
			server: server{
				cookieToTEEPubKey: make(map[string]*jose.JSONWebKey),
				cookieToNonce:     make(map[string]string),
			},
			req: testRequest{
				method: http.MethodPost,
				body:   "",
				cookie: &http.Cookie{Name: sessionIDCookieName, Value: "someValue"},
			},
			wantStatus: http.StatusBadRequest,
		},
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
			if tc.wantStatus == http.StatusOK {
				assert.Len(tc.server.cookieToTEEPubKey, 1)
				assert.Len(tc.server.cookieToNonce, 1)
				return
			}
			assert.Len(tc.server.cookieToTEEPubKey, 0)
			assert.Len(tc.server.cookieToNonce, 0)
		})
	}
}

func TestUnimplemented(t *testing.T) {
	testCases := []string{
		"/kbs/v0/attestation-policy",
		"/kbs/v0/token-certificate-chain",
	}

	for _, path := range testCases {
		t.Run(path, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			server := httptest.NewServer((&server{}).newHandler())
			defer server.Close()
			client := server.Client()

			req, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
			require.NoError(err)
			resp, err := client.Do(req)
			require.NoError(err)

			assert.Equal(http.StatusNotImplemented, resp.StatusCode)
		})
	}
}

func genPrivKey() *rsa.PrivateKey {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return privKey
}

func genJWK() *jose.JSONWebKey {
	privKey := genPrivKey()
	jwk := jose.JSONWebKey{Key: privKey.Public(), KeyID: "someKeyID"}
	return &jwk
}

type testRequest struct {
	method string
	body   string
	cookie *http.Cookie
}

func mustVal[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
