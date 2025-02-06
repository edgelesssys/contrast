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
	// Nonce
	nonce []byte
	// AdditionalData
	associatedData []byte
}

// symmetricEncryptRaw returns the encrypted plaintext based on the symmetric options and encryption key handed in.
func symmetricEncryptRaw(encKey, plaintext []byte, opts symOpts) ([]byte, error) {
	aesCipher, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return nil, err
	}
	if opts.nonce == nil {
		randomNonce, err := crypto.GenerateRandomBytes(gcm.NonceSize())
		if err != nil {
			return nil, err
		}
		opts.nonce = randomNonce
	}
	ciphertext := gcm.Seal(nil, opts.nonce, plaintext, opts.associatedData)
	return append(opts.nonce, ciphertext...), nil
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
	opts.nonce = ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, opts.nonce, ciphertext, opts.associatedData)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
