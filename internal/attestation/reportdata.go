// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package attestation

import "crypto/sha512"

func ConstructReportData(publicKey []byte, nonce []byte) [64]byte {
	return sha512.Sum512(append(publicKey, nonce...))
}
