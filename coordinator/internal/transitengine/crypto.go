// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package transitengine

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/edgelesssys/contrast/internal/crypto"
)

// symmetricEncryptRaw returns a ciphertextContainer, based on the encryption key and associatedData handed in.
func symmetricEncryptRaw(encKey, plaintext b64Plaintext, associatedData []byte) (ciphertextContainer, error) {
	aesCipher, err := aes.NewCipher(encKey)
	if err != nil {
		return ciphertextContainer{}, err
	}
	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return ciphertextContainer{}, err
	}
	nonce, err := crypto.GenerateRandomBytes(gcm.NonceSize())
	if err != nil {
		return ciphertextContainer{}, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, associatedData)
	return ciphertextContainer{
		nonce:      nonce,
		ciphertext: ciphertext,
	}, nil
}

// symmetricDecryptRaw extracts the nonce and returns the decrypted ciphertext based on encryption keys handed in,
// if the associatedData is valid.
func symmetricDecryptRaw(decKey []byte, ciphertextContainer ciphertextContainer, associatedData []byte) (b64Plaintext, error) {
	aesCipher, err := aes.NewCipher(decKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, ciphertextContainer.nonce, ciphertextContainer.ciphertext, associatedData)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
