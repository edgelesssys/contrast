// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package transitengine

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
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
		fakeStateGuard, err := newTestGuard()
		require.NoError(t, err)
		mux := newMockTransitEngineMux(fakeStateGuard)

		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				var ciphertext, receivedPlaintext string
				t.Run("encryption request handling", func(t *testing.T) {
					require := require.New(t)
					jsonBody, err := json.Marshal(tc.encryptionInput)
					require.NoError(err)

					req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/v1/transit/encrypt/"+tc.name, bytes.NewReader(jsonBody))
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

					decryptReq := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/v1/transit/decrypt/"+tc.name, bytes.NewReader(decryptReqBody))
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

func TestAdversarialDecryptInputs(t *testing.T) {
	testCases := map[string]struct {
		payload  string
		wantCode int
	}{
		"empty": {
			payload:  ``,
			wantCode: http.StatusBadRequest,
		},
		"no fields": {
			payload:  `{}`,
			wantCode: http.StatusBadRequest,
		},
		"truncated identifier": {
			payload:  `{"ciphertext":"vaul","associated_data":""}`,
			wantCode: http.StatusBadRequest,
		},
		"truncated nonce": {
			payload:  `{"ciphertext":"vault:v1:AAAA","associated_data":""}`,
			wantCode: http.StatusBadRequest,
		},
		"random data": {
			payload:  `{"ciphertext":"vault:v1:DEOLYTbWcdjNqFjrfb/heTN7P1LohVS8KpGYisecIs2O2FrjevU3zrJHPTe6biaalMa2xphSNt6JFvWpCSsW4svHe2r0myjnq05Zbi+h37GIS2FYPrRDnoglsRHdJ+rH+D0MBJON0WQnVE9qbSDI4P9cjZZXVm7lx2VAqu3ioWw=","associated_data":""}`,
			wantCode: http.StatusBadRequest,
		},
		//
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			fakeStateGuard, err := newTestGuard()
			require.NoError(err)
			mux := newMockTransitEngineMux(fakeStateGuard)

			decryptReq := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/v1/transit/decrypt/foo", bytes.NewReader([]byte(tc.payload)))
			decryptReq.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, decryptReq)
			res := rec.Result()
			t.Cleanup(func() { _ = res.Body.Close() })
			body, err := io.ReadAll(res.Body)
			require.NoError(err)
			require.Equal(tc.wantCode, res.StatusCode, string(body))
		})
	}
}

type fakeStateGuard struct {
	state *stateguard.State
}

func newTestGuard() (*fakeStateGuard, error) {
	salt := make([]byte, constants.SecretSeedSaltSize)
	secretSeed := make([]byte, constants.SecretSeedSize)
	seedEngine, err := seedengine.New(secretSeed, salt)
	if err != nil {
		return nil, err
	}
	fakeState := stateguard.NewStateForTest(seedEngine, nil, nil, nil)

	guard := &fakeStateGuard{
		state: fakeState,
	}
	return guard, nil
}

func (f *fakeStateGuard) GetState(context.Context) (*stateguard.State, error) {
	return f.state, nil
}

func newMockTransitEngineMux(guard stateGuard) *http.ServeMux {
	mux := http.NewServeMux()
	logger := slog.New(slog.DiscardHandler)
	mux.Handle("/v1/transit/encrypt/{name}", getEncryptHandler(guard, logger))
	mux.Handle("/v1/transit/decrypt/{name}", getDecryptHandler(guard, logger))
	return mux
}
