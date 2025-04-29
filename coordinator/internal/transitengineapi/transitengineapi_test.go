// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package transitengine

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/edgelesssys/contrast/coordinator/internal/authority"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/seedengine"
	"github.com/stretchr/testify/require"
)

func TestTransitAPICyclic(t *testing.T) {
	type encryptionInput struct {
		Plaintext      string `json:"plaintext"`
		AssociatedData string `json:"associated_data,omitempty"`
		Version        int    `json:"key_version"`
	}
	testCases := map[string]struct {
		name            string
		encryptionInput encryptionInput
		expStatus       int
	}{
		"positive": {
			name: "autounseal",
			encryptionInput: encryptionInput{
				Plaintext:      "vpIhKQhFuGwLv5B/XLYr960uZQ==",
				AssociatedData: "Xv0nZLWkSan+vdWrH2LrGP8TU/Qg1+ZX7vldWMbxKTk=",
				Version:        2,
			},
			expStatus: 200,
		},
		"special char name": {
			name: "thi$$hoU_ld+*work",
			encryptionInput: encryptionInput{
				Plaintext:      "lT3rQGMlxq680DdSKfIYYcfyCfMnP9ikxaO5b0mGRKRl4qNL3W9xkSW3QmaMwozCRfNMZhhDCbYokn6KEiGotlVInKt66QjBgXR2Nk9hIcez0LYt8W5pxD0lwTxC",
				AssociatedData: "HribV+LFZspJpAauFf643A1HKbj1VlQWVhAKFDJqdZg=",
				Version:        2000,
			},
			expStatus: 200,
		},
		"failed not found empty name": {
			name: "",
			encryptionInput: encryptionInput{
				Plaintext:      "vpIh",
				AssociatedData: "vpIh",
			},
			expStatus: 404,
		},
		"failed not found name with forward slash": {
			name: "wrong/url",
			encryptionInput: encryptionInput{
				Plaintext:      "vpIh",
				AssociatedData: "vpIh",
			},
			expStatus: 404,
		},
		"failed bad request no base64 plaintext": {
			name: "autounseal",
			encryptionInput: encryptionInput{
				Plaintext:      "thi$$hoU_ld+*notwork",
				AssociatedData: "vpIh",
			},
			expStatus: 400,
		},
		"failed bad request no base64 additional data": {
			name: "autounseal",
			encryptionInput: encryptionInput{
				Plaintext:      "HribV+LFZspJpAauFf643A1HKbj1VlQWVhAKFDJqdZg=",
				AssociatedData: "thi$$hoU_ld+*notwork",
			},
			expStatus: 400,
		},
		"failed bad request negative key version": {
			name: "thi$$hoU_ld+*work",
			encryptionInput: encryptionInput{
				Plaintext:      "lT3rQGMlxq680DdSKfIYYcfyCfMnP9ikxaO5b0mGRKRl4qNL3W9xkSW3QmaMwozCRfNMZhhDCbYokn6KEiGotlVInKt66QjBgXR2Nk9hIcez0LYt8W5pxD0lwTxC",
				AssociatedData: "HribV+LFZspJpAauFf643A1HKbj1VlQWVhAKFDJqdZg=",
				Version:        -2000,
			},
			expStatus: 400,
		},
	}

	t.Run("encrypt-decrypt handler", func(t *testing.T) {
		fakeStateAuthority, err := newFakeSeedEngineAuthority()
		require.NoError(t, err)
		mux := newMockTransitEngineMux(fakeStateAuthority)

		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				var ciphertext, receivedPlaintext string
				t.Run("encryption request handling", func(t *testing.T) {
					require := require.New(t)
					jsonBody, err := json.Marshal(tc.encryptionInput)
					require.NoError(err)

					req := httptest.NewRequest(http.MethodPut, "/v1/transit/encrypt/"+tc.name, bytes.NewReader(jsonBody))
					req.Header.Set("Content-Type", "application/json")

					rec := httptest.NewRecorder()
					mux.ServeHTTP(rec, req)
					res := rec.Result()
					require.Equal(tc.expStatus, res.StatusCode)
					if tc.expStatus != http.StatusOK {
						ciphertext = "vault:v" + strconv.Itoa(tc.encryptionInput.Version) + tc.encryptionInput.Plaintext
						return
					}

					var encRespBody map[string]map[string]string
					require.NoError(json.NewDecoder(res.Body).Decode(&encRespBody))
					defer res.Body.Close()

					data, exist := encRespBody["data"]
					require.True(exist)
					ciphertext, exist = data["ciphertext"]
					require.True(exist)
				})

				t.Run("decryption request handling", func(t *testing.T) {
					require := require.New(t)
					decryptReqBody, err := json.Marshal(
						map[string]string{
							"ciphertext":      ciphertext,
							"associated_data": tc.encryptionInput.AssociatedData,
						})
					require.NoError(err)

					decryptReq := httptest.NewRequest(http.MethodPut, "/v1/transit/decrypt/"+tc.name, bytes.NewReader(decryptReqBody))
					decryptReq.Header.Set("Content-Type", "application/json")

					rec := httptest.NewRecorder()
					mux.ServeHTTP(rec, decryptReq)
					res := rec.Result()
					require.Equal(tc.expStatus, res.StatusCode)
					if tc.expStatus != http.StatusOK {
						return
					}

					var decRespBody map[string]map[string]string
					require.NoError(json.NewDecoder(res.Body).Decode(&decRespBody))
					defer res.Body.Close()

					data, exist := decRespBody["data"]
					require.True(exist)
					receivedPlaintext, exist = data["plaintext"]
					require.True(exist)
					require.Equal(tc.encryptionInput.Plaintext, receivedPlaintext, "Unexpected received plaintext after cycling handler functions")
				})
			})
		}
	})
}

type fakeStateAuthority struct {
	state *authority.State
}

func newFakeSeedEngineAuthority() (*fakeStateAuthority, error) {
	salt := make([]byte, constants.SecretSeedSaltSize)
	secretSeed := make([]byte, constants.SecretSeedSize)
	seedEngine, err := seedengine.New(secretSeed, salt)
	if err != nil {
		return nil, err
	}
	fakeState := authority.NewState(seedEngine, nil, nil, nil)

	authority := &fakeStateAuthority{
		state: fakeState,
	}
	return authority, nil
}

func (f *fakeStateAuthority) GetState() (*authority.State, error) {
	return f.state, nil
}

func newMockTransitEngineMux(authority stateAuthority) *http.ServeMux {
	mux := http.NewServeMux()
	logger := slog.New(slog.DiscardHandler)
	mux.Handle("/v1/transit/encrypt/{name}", getEncryptHandler(authority, logger))
	mux.Handle("/v1/transit/decrypt/{name}", getDecryptHandler(authority, logger))
	return mux
}
