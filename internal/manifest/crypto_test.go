// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"testing"

	"github.com/edgelesssys/contrast/internal/userapi"
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
						keys[i] = newKey(t, b)
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

		rightKey := newKey(t, 2048)
		wrongKey := newKey(t, 2048)

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

	key := newKey(t, 2048)

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

func newKey(t *testing.T, bits int) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err)
	require.NoError(t, key.Validate())

	t.Logf("generated key: %q", HexString(x509.MarshalPKCS1PrivateKey(key)))
	return key
}
