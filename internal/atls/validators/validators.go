// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// package validators defines the Validator interface and helpers for working with validators.
package validators

import (
	"context"
	"encoding/asn1"
	"errors"
)

// ErrOIDNotSupported is returned by a validator when it doesn't understand the OID provided as input.
var ErrOIDNotSupported = errors.New("OID not supported")

// Validator is able to validate an attestation document.
//
// Validators are encouraged to implement the fmt.Stringer interface to improve logging.
type Validator interface {
	// Validate validates an attestation doc and returns an error if validation failed.
	//
	// Implementations should first check whether they understand the given OID. If they don't,
	// they should return ErrOIDNotSupported.
	//
	// If validation passes, the validator guarantees that the given reportData was present in the
	// attestation document.
	Validate(ctx context.Context, oid asn1.ObjectIdentifier, attDoc []byte, reportData []byte) error
}

// ValidatorFunc creates a validator from a func.
type ValidatorFunc func(context.Context, asn1.ObjectIdentifier, []byte, []byte) error

// Validate calls the adapted func to implement Validator.Validate.
func (f ValidatorFunc) Validate(ctx context.Context, oid asn1.ObjectIdentifier, attDoc []byte, reportData []byte) error {
	return f(ctx, oid, attDoc, reportData)
}

// NoValidation skips validation of the server's attestation document.
func NoValidation() []Validator {
	return []Validator{}
}
