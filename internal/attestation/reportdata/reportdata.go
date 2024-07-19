// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package reportdata

import "crypto/sha512"

// Construct attestation report data from public key and nonce.
func Construct(publicKey []byte, nonce []byte) [64]byte {
	return sha512.Sum512(append(publicKey, nonce...))
}
