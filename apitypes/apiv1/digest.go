// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apiv1

import (
	"crypto/sha256"
	"fmt"
)

const (
	// NonceSize is the size of the nonces used by this API version.
	NonceSize = 32
	// DigestSize is the size of the digests bound by this API version, e.g. transition hashes.
	DigestSize = sha256.Size
)

// constructDigest builds the digest all report data values and signature inputs of this API version share.
//
//	digest = sha256(sha256(label) || field_1 || ... || field_n)
//
// The label names the construction and its version, so that a digest made for one purpose isn't accepted for another.
// Callers must only pass fields of a fixed length, which is what makes the concatenation unambiguous without length
// prefixes. Variable-length values enter as their SHA-256 digest, see [hashed].
func constructDigest(label string, fields ...[]byte) [sha256.Size]byte {
	h := sha256.New()
	h.Write(hashed([]byte(label)))
	for _, field := range fields {
		h.Write(field)
	}
	return [sha256.Size]byte(h.Sum(nil))
}

// hashed returns the SHA-256 digest of a variable-length value. An absent value is hashed as the empty string.
func hashed(value []byte) []byte {
	digest := sha256.Sum256(value)
	return digest[:]
}

// fixed checks that a field has the fixed length the digest construction expects it to have.
func fixed(name string, value []byte, size int) error {
	if len(value) != size {
		return fmt.Errorf("invalid %s length: got %d, expected %d", name, len(value), size)
	}
	return nil
}

// asReportData places a digest in the first half of the report data and leaves the rest zero.
func asReportData(digest [sha256.Size]byte) [ReportDataSize]byte {
	var reportData [ReportDataSize]byte
	copy(reportData[:], digest[:])
	return reportData
}
