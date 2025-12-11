// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package transitengine

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/edgelesssys/contrast/internal/cryptohelpers"
)

// ciphertextContainer describes a base64-encoded ciphertext prepended with the nonce and specified key version.
type ciphertextContainer struct {
	nonce      []byte
	ciphertext []byte
	keyVersion uint32
}

// symmetricEncryptRaw returns a ciphertextContainer, based on the encryption key and associatedData handed in.
func symmetricEncryptRaw(encKey, plaintext []byte, associatedData []byte) (ciphertextContainer, error) {
	aesCipher, err := aes.NewCipher(encKey)
	if err != nil {
		return ciphertextContainer{}, err
	}
	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return ciphertextContainer{}, err
	}
	nonce, err := cryptohelpers.GenerateRandomBytes(gcm.NonceSize())
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
func symmetricDecryptRaw(decKey []byte, ciphertextContainer ciphertextContainer, associatedData []byte) ([]byte, error) {
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

// UnmarshalJSON umarshalls a json string to a ciphertextContainer holding the version prefix,
// decoded base64 nonce and ciphertext.
func (c *ciphertextContainer) UnmarshalJSON(data []byte) error {
	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}
	// Split "vault:vX:base64" format
	parts := strings.SplitN(encoded, ":", 3)
	if len(parts) < 3 {
		return fmt.Errorf("invalid ciphertext format")
	}
	version, err := extractVersion(parts[1])
	if err != nil {
		return fmt.Errorf("ciphertext version: %w", err)
	}
	c.keyVersion = version
	fullCiphertext, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("decoding ciphertext: %w", err)
	}
	c.nonce = fullCiphertext[:aesGCMNonceSize]
	c.ciphertext = fullCiphertext[aesGCMNonceSize:]
	return nil
}

// MarshalJSON marshalls a ciphertextContainer to a json string.
func (c ciphertextContainer) MarshalJSON() ([]byte, error) {
	fullCiphertext := append(c.nonce, c.ciphertext...)
	encodedfullCiphertext := base64.StdEncoding.EncodeToString(fullCiphertext)
	// Convert to "vault:vX:base64" format
	versioned := fmt.Sprintf("vault:v%d:%s", c.keyVersion, encodedfullCiphertext)
	return json.Marshal(versioned)
}
