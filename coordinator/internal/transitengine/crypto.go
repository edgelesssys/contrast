// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package transitengine

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/edgelesssys/contrast/internal/crypto"
)

const (
	// aesGCMNonceSize specifies the default nonce size in bytes used in AES GCM.
	aesGCMNonceSize = 12
	// aesGCMKeySize specifies the default key size in bytes AES GCM.
	aesGCMKeySize = 16
)

// symOpts holds parameters related to the performed symmetric encryption, specifyable as http request parameters.
type symOpts struct {
	// Convergent TODO(jmxzo): add parameter support
	convergent bool
	// Nonce
	nonce []byte
	// AdditionalData
	additionalData []byte //nolint
}

// symmetricEncryptRaw returns the encrypted plaintext based on the symmetric options and encryption key handed in.
func symmetricEncryptRaw(encKey, plaintext []byte, _ symOpts) ([]byte, error) {
	aesCipher, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return nil, err
	}
	nonce, err := crypto.GenerateRandomBytes(12)
	if err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// symmetricDecryptRaw returns the decrypted ciphertext based on the symmetric options and encryption keys handed in.
func symmetricDecryptRaw(decKey, ciphertext []byte, opts symOpts) ([]byte, error) {
	aesCipher, err := aes.NewCipher(decKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, opts.nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
