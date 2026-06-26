// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// package validators defines the Validator interface and helpers for working with validators.
package validators

import (
	"context"
	"encoding/asn1"
	"fmt"
)

// Validator is able to validate an attestation document.
type Validator interface {
	// Supports returns true if this validator understands attestation docs with the given OID.
	Supports(oid asn1.ObjectIdentifier) bool

	// Validate validates an attestation doc and returns an error if validation failed.
	//
	// If validation passes, the validator guarantees that the given reportData was present in the
	// attestation document.
	Validate(ctx context.Context, attDoc []byte, reportData []byte) error

	// Stringer allows better logging of validation results.
	fmt.Stringer
}

// NoValidation skips validation of the server's attestation document.
func NoValidation() []Validator {
	return []Validator{}
}
