// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package transitengine

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCryptoCyclic(t *testing.T) {
	testCases := map[string]struct {
		key            string
		plaintext      string
		associatedData string
	}{
		"plaintext 1": {
			key:            "H0ltoqZhwB/h1/HqIPWr5rC9Xt/sGkJ3UDU10vYplGw=",
			plaintext:      "rWk4fnCShUgfxjMShRHS5Q==",
			associatedData: "VKpPXX5vwsZUPdFTQOTIdg==",
		},
		"plaintext 2": {
			key:            "RV30FnNmnE2b8hUAL4P89tsqF/346nODNTXdYMrOuA0=",
			plaintext:      "ZSZ/iGeN22qnQrvkogjvVnZG/3zhCboO/DC8Gl/GAFBRXYMh4112FA==",
			associatedData: "c0JDPPZH0yrHVrl+x8e9JE0wgxrjCBk9Z2Vpi72+Fvy2wUMVPZ2i/A==",
		},
		"plaintext 3": {
			key:            "vMYJT/IsiWCOX5/8cAcP7eEM7G4ZRwMNQtauHORAXY4=",
			plaintext:      "X3pjwqhM4wMM/OFTpbZsOECwtZ4TZs0ZCptD/kg=",
			associatedData: "f6kf/DPKlb4ig6JJ7Ls0a+EupP3QSrmkj/bTkIuaW6tm+7z2Ugd4cg==",
		},
		"plaintext 4": {
			key:            "BlWwHNYdKgELBFXuS1BIkccaRY6DHPRmIHWAr6gje+Q=",
			plaintext:      "XXjgvMHQ3CASYjqHhq1i5rplfNHS",
			associatedData: "23HOhX2Pqm7qWwa8X/lSIQKRQ+s=",
		},
	}

	t.Run("cyclic crypto test", func(t *testing.T) {
		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				var ciphertextContainer ciphertextContainer
				var recPlaintextBytes []byte

				keyBytes, err := base64.StdEncoding.DecodeString(tc.key)
				require.NoError(t, err)
				plaintextBytes, err := base64.StdEncoding.DecodeString(tc.plaintext)
				require.NoError(t, err)
				associatedDataBytes, err := base64.StdEncoding.DecodeString(tc.associatedData)
				require.NoError(t, err)
				t.Run("encrypt", func(t *testing.T) {
					require := require.New(t)
					var err error
					ciphertextContainer, err = symmetricEncryptRaw(keyBytes, plaintextBytes, associatedDataBytes)
					require.NoError(err, "encryption")
				})

				t.Run("decrypt", func(t *testing.T) {
					require := require.New(t)
					var err error
					recPlaintextBytes, err = symmetricDecryptRaw(keyBytes, ciphertextContainer, associatedDataBytes)
					require.NoError(err, "decryption")
				})
				require.Equal(t, plaintextBytes, recPlaintextBytes)
			})
		}
	})
}
