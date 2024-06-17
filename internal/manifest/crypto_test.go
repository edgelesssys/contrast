// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"testing"

	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptSingleKey(t *testing.T) {
	require := require.New(t)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(err)

	seed := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	pubKeyHex := MarshalSeedShareOwnerKey(&key.PublicKey)

	seedShares, err := EncryptSeedShares(seed, []HexString{pubKeyHex})
	require.NoError(err)
	require.Len(seedShares, 1)

	decryptedSeedShare, err := DecryptSeedShare(key, seedShares[0])
	require.NoError(err)

	require.Equal(seed, decryptedSeedShare)

	// Decrypting with another key should fail.

	key2, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(err)

	_, err = DecryptSeedShare(key2, seedShares[0])
	require.Error(err)
}

func TestEncryptDecrypt_WrongLabel(t *testing.T) {
	require := require.New(t)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(err)

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
