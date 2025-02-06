// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package transitengine

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/coordinator/internal/seedengine"
	"github.com/stretchr/testify/require"
)

func TestCryptoAPICyclic(t *testing.T) {
	data := []map[string]string{
		{"Plaintext": "vpIhKQhFuGwLv5B/XLYr960uZQ==", "Ciphertext": "vault:v1:d01M+wZYaG9LFnuc18s8oh6PuVFw3+7DBX4LkXXQ6d64DvmQt6qwjj1MHmA88UE="},
		{"Plaintext": "O/XShnapt5hNMCZnP+4BZjH84CcWAwhxOUnGwKH7ja1ZYsZdyZrGeLT4EZtA1vWey04bAsi+viGmpYO98YbkCvSn7HZvglLh2DMv3Ach9SP2qjWw0NBa2rrfToI1dsE=", "Ciphertext": "vault:v1:onRtviw8oJv5cY4EngPYySbYlAYgEqWXk5WbD2X+jpRDR5d87Y2qE0Otc4KUKja8LYVb1zB1P40yWTgNb7srG4D7kuxlRFFbtkYaCAC3bLbz+QuumLFQd9RbsN7avbLtEmT8nay/1qvvp2e/MDv5oPoDIT0vzHorVI40"},
		{"Plaintext": "lT3rQGMlxq680DdSKfIYYcfyCfMnP9ikxaO5b0mGRKRl4qNL3W9xkSW3QmaMwozCRfNMZhhDCbYokn6KEiGotlVInKt66QjBgXR2Nk9hIcez0LYt8W5pxD0lwTxC", "Ciphertext": "vault:v1:taQ++Amvoj0G1V+OQb0PjldLh5BRXmAhwlO38LRjajVIuHTEa69kfytU3mMaFEpG5JNVg3Cq6vSWH58n1NEmM6WDV79q/hxzaji68joq9uQeoyH3To9iBoRHE2T7jbxUvYLeBwgzFvM/YNk5EaFpNkfIwwxzNZ5gmQ=="},
		{"Plaintext": "WC0sC1KtNw76hGVQTpeFNtPg94tJc64dE3rf0mhENsBMLhWmYinA99YbIGx0gSQEOkOsR1sPgSnEmxTvycQdNA==", "Ciphertext": "vault:v1:vjeAbnpFxh+atxQsk6OeumH6irWPAuRbim2UQ8ggNGQFpB4wnYxjUiydGgYixZ6x1Ad+STfbjwxLvij5ZpmyMxFsmrYZIQKCYE48mVUNI+VJa87zuMabQCMClOs="},
		{"Plaintext": "iZo0IezTmQ6Ms1GJUbbY4nrsRydO31b6xuqlJwi+R9xLt1K9uaI8ZiuInXc92qpulYNaAWAiBmNNghKM0dpAPdSXwc93+YCT1Zm2i1cuW7H6Uz5tL7E=", "Ciphertext": "vault:v1:qqjtmjUKfzoAb3VeUgWjbRDdYu3K/cdmi7sEVcKPRiOdbY5OyuQrNQHtYZ/mje8hcPyHnmgDJEDOpjLhQgUG6yoYamqksut//lv7DDbKYbzroro5BRiCqQjhfqnhmna79maV5okq8zI5YZBoSSn36ivu"},
		{"Plaintext": "FLiOXWBy8oVETtiNzfw0rbMgCfa3DVSfKL4GhR86EcluUe78nLiDxt0HtP5vTwaaz7mvLXu2nOtsdlz8kYY1YrZCLLNlYzjm/vYe++CcH/x+fDKepJem5le0BCsdog==", "Ciphertext": "vault:v1:0qp8vsJ+JFf5m5HekxQeUw0+gj/NdoDcmy7ExSw0G7PB1RBQ80T+TUMjSvmmgu02eQ2oCKEkfFMolfNt1zq+sZkLQuTLpbW8p+Vd8ALPGdyyD20MIb2ez6dm9nMM4jiXL2FkfARuHcHoY3/LCBQVLE+kkJLAdwze/Z4="},
		{"Plaintext": "Xqv9SzcpNK99JF1I7xRAJ0FOkzka", "Ciphertext": "vault:v1:N431rZS6bcuGDDJ8Jh93yvih4oc5wHGtz02M2vFQ5IioIlZFqfv29nDImWNaZUW7Zw=="},
		{"Plaintext": "6HQ035OxE30=", "Ciphertext": "vault:v1:ZUuU9vMiCa3CE7PvmYlJYhHQOJm2/Rk6xuUA7DSJu4ExttPl"},
		{"Plaintext": "yqUBQzznRjbXMxhQkwo5q2Az3/6nvgRQ86uffx8ZqT7rufplfhJz+xDfvi4EOw==", "Ciphertext": "vault:v1:fpjOfUK8hZEk6DmcH715zdyTMAqyW9ymAMxQFgoMR2Q5639NYCA3rbrR4OKfGwCqO7UrBg3LeFksYBPhiO4pbX6dHXGYIfjDdgo="},
	}
	t.Run("encrypt-decrypt handler", func(t *testing.T) {
		mux := NewTransitEngineAPI(&fakeSeedEngineAuthority{}, slog.Default())
		for _, entry := range data {
			t.Run("cyclic handler function testing", func(t *testing.T) {
				var ciphertext, receivedPlaintext string
				t.Run("encryption request handling", func(t *testing.T) {
					require := require.New(t)
					jsonBody, err := createReqBodyJSON("plaintext", entry["Plaintext"])
					require.NoError(err)
					req := httptest.NewRequest(http.MethodPut, "/v1/transit/encrypt/autounseal", bytes.NewReader(jsonBody))
					req.Header.Set("Content-Type", "application/json")
					require.NoError(err)
					encRespBody := receiveResponseBody(t, mux, req)
					require.NoError(err)
					data, _ := encRespBody["data"].(map[string]any)
					ciphertext, _ = data["ciphertext"].(string)
				})

				t.Run("decryption request handling", func(t *testing.T) {
					require := require.New(t)
					decryptReqBody, err := createReqBodyJSON("ciphertext", ciphertext)
					require.NoError(err)
					decryptReq := httptest.NewRequest(http.MethodPut, "/v1/transit/decrypt/autounseal", bytes.NewReader(decryptReqBody))
					decryptReq.Header.Set("Content-Type", "application/json")
					decRespBody := receiveResponseBody(t, mux, decryptReq)
					data, _ := decRespBody["data"].(map[string]any)
					receivedPlaintext, _ = data["plaintext"].(string)
					require.Equal(entry["Plaintext"], receivedPlaintext, "Unexpected received plaintext after cycling handler functions")
				})
			})
		}
	})
}

func receiveResponseBody(t *testing.T, mux *http.ServeMux, req *http.Request) map[string]any {
	require := require.New(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	require.Equal(http.StatusOK, res.StatusCode)
	respBody, err := io.ReadAll(res.Body)
	require.NoError(err)
	var respData map[string]any
	err = json.Unmarshal(respBody, &respData)
	require.NoError(err)
	return respData
}

func createReqBodyJSON(bodyParameter, value string) ([]byte, error) {
	body := map[string]string{
		bodyParameter: value,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return jsonBody, nil
}

type fakeSeedEngineAuthority struct{}

func (f *fakeSeedEngineAuthority) GetSeedEngine() (seedEngine *seedengine.SeedEngine, err error) {
	salt := make([]byte, 32)       // 32-byte salt initialized with zeros
	secretSeed := make([]byte, 32) // 32-byte secret seed initialized with zeros
	seedEngine, err = seedengine.New(secretSeed, salt)
	if err != nil {
		return nil, err
	}
	return seedEngine, nil
}
