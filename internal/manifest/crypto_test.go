// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptSingleKey(t *testing.T) {
	bits := []int{2048, 4096}
	seeds := [][]byte{{}, {1, 2, 3, 4, 5, 6, 7, 8}}

	for _, b := range bits {
		for numKeys := range 3 {
			for _, seed := range seeds {
				name := fmt.Sprintf("bits=%d numKeys=%d seed=[%d]byte", b, numKeys, len(seed))
				t.Run(name, func(t *testing.T) {
					require := require.New(t)
					keys := make([]*rsa.PrivateKey, numKeys)
					pubKeys := make([]HexString, numKeys)
					for i := range numKeys {
						keys[i] = getTestKey(t, b, i)
						pubKeys[i] = MarshalSeedShareOwnerKey(&keys[i].PublicKey)
					}

					seedShares, err := EncryptSeedShares(seed, pubKeys)
					require.NoError(err)
					require.Len(seedShares, numKeys)

					for i := range numKeys {
						decryptedSeedShare, err := DecryptSeedShare(keys[i], seedShares[i])
						require.NoError(err)

						require.Equal(seed, decryptedSeedShare)
					}
				})
			}
		}
	}

	t.Run("decrypting with an unrelated key should fail", func(t *testing.T) {
		require := require.New(t)

		rightKey := getTestKey(t, 2048, 1)
		wrongKey := getTestKey(t, 2048, 2)

		seed := []byte{1, 2, 3, 4, 5, 6, 7, 8}

		pubKeyHex := MarshalSeedShareOwnerKey(&rightKey.PublicKey)

		seedShares, err := EncryptSeedShares(seed, []HexString{pubKeyHex})
		require.NoError(err)
		require.Len(seedShares, 1)

		decryptedSeedShare, err := DecryptSeedShare(wrongKey, seedShares[0])
		require.Error(err)
		require.Nil(decryptedSeedShare)
	})
}

func TestDecryptSeedShare_WrongLabel(t *testing.T) {
	require := require.New(t)

	key := getTestKey(t, 2048, 1)

	seed := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	pubKeyHex := MarshalSeedShareOwnerKey(&key.PublicKey)

	cipherText, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &key.PublicKey, seed, []byte("this-label-is-wrong-for-contrast-seeds"))
	require.NoError(err)
	seedShare := &userapi.SeedShare{
		EncryptedSeed: cipherText,
		PublicKey:     pubKeyHex.String(),
	}

	_, err = DecryptSeedShare(key, seedShare)
	require.Error(err)
}

func TestSeedShareKeyParseMarshal(t *testing.T) {
	key := testkeys.RSA(t)

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(key)
	keyData := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes})

	pubHexStr := MarshalSeedShareOwnerKey(&key.PublicKey)

	_, err := ParseSeedShareOwnerKey(pubHexStr)
	require.NoError(t, err)

	publicKeyHexStr, err := ExtractSeedshareOwnerPublicKey(keyData)
	require.NoError(t, err)

	publicKeyBytes, err := publicKeyHexStr.Bytes()
	require.NoError(t, err)
	publicKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})
	publicKeyHexStrReparsed, err := ExtractSeedshareOwnerPublicKey(publicKeyPem)
	require.NoError(t, err)
	assert.Equal(t, publicKeyHexStr, publicKeyHexStrReparsed)
}

func TestWorkloadOwnerKeyParseMarshal(t *testing.T) {
	key := testkeys.ECDSA(t)

	privateKeyBytes, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	keyData := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes})
	privateKey, err := ParseWorkloadOwnerPrivateKey(keyData)
	require.NoError(t, err)

	keyDigest := HashWorkloadOwnerKey(&privateKey.PublicKey)
	assert.Len(t, keyDigest, 64)

	publicKeyBytes, err := ExtractWorkloadOwnerPublicKey(keyData)
	require.NoError(t, err)

	publicKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})
	publicKeyBytesReparsed, err := ExtractWorkloadOwnerPublicKey(publicKeyPem)
	require.NoError(t, err)
	assert.Equal(t, publicKeyBytes, publicKeyBytesReparsed)
}

func getTestKey(t *testing.T, keyLen int, id int) *rsa.PrivateKey {
	t.Helper()

	var keyStr testkeys.EncodedKeyString
	switch keyLen {
	case 2048:
		if id >= len(testkeys.RSA2048Keys) {
			t.Fatalf("test key %d [%d bit] not found", id, keyLen)
		}
		keyStr = testkeys.RSA2048Keys[id]
	case 4096:
		if id >= len(testkeys.RSA4096Keys) {
			t.Fatalf("test key %d [%d bit] not found", id, keyLen)
		}
		keyStr = testkeys.RSA4096Keys[id]
	default:
		t.Fatalf("unsupported key length: %d", keyLen)
	}

	key := testkeys.New[rsa.PrivateKey](t, keyStr)
	return key
}
