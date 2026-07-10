// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// package validators defines the Validator interface and helpers for working with validators.
package validators

import (
	"context"
	"encoding/asn1"
	"errors"
	"fmt"
	"strings"
)

// ErrOIDNotSupported is returned by a validator when it doesn't understand the OID provided as input.
var ErrOIDNotSupported = errors.New("OID not supported")

// Validator is able to validate an attestation document.
//
// Validators are required to implement the fmt.Stringer interface to improve logging.
type Validator interface {
	// Validate validates an attestation doc and returns an error if validation failed.
	//
	// Implementations should first check whether they understand the given OID. If they don't,
	// they should return ErrOIDNotSupported.
	//
	// If validation passes, the validator guarantees that the given reportData was present in the
	// attestation document.
	Validate(ctx context.Context, oid asn1.ObjectIdentifier, attDoc []byte, reportData []byte) error

	fmt.Stringer
}

// ValidatorFunc creates a validator from a func.
type ValidatorFunc func(context.Context, asn1.ObjectIdentifier, []byte, []byte) error

// Validate calls the adapted func to implement Validator.Validate.
func (f ValidatorFunc) Validate(ctx context.Context, oid asn1.ObjectIdentifier, attDoc []byte, reportData []byte) error {
	return f(ctx, oid, attDoc, reportData)
}

func (f ValidatorFunc) String() string {
	return "ValidatorFunc"
}

type anyOf struct {
	vs []Validator
}

func (a *anyOf) Validate(ctx context.Context, oid asn1.ObjectIdentifier, attDoc []byte, reportData []byte) error {
	interestingErrors := make([]error, 0, len(a.vs))
	for i, v := range a.vs {
		err := v.Validate(ctx, oid, attDoc, reportData)
		if err == nil {
			return nil
		}
		// A bunch of "unsupported" errors would clutter the output, only add the interesting ones.
		if !errors.Is(err, ErrOIDNotSupported) {
			interestingErrors = append(interestingErrors, fmt.Errorf("sub-validator %d: %w", i, err))
		}
	}
	// No validator passed, let's decide what to report back.
	if len(interestingErrors) == 0 {
		// If no error was added to the list, all errors were "not supported". Return that to
		// the caller.
		return ErrOIDNotSupported
	}
	// Bundle all interesting errors into one.
	return errors.Join(interestingErrors...)
}

func (a *anyOf) String() string {
	b := strings.Builder{}
	b.WriteString("Any(")
	for i, v := range a.vs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(v.String())
	}
	b.WriteString(")")
	return b.String()
}

// Any creates a Validator that passes if one of the input Validators passes.
//
// The Validators are tried in order, until one succeeds or no more are left.
// The combined Validator supports all OIDs that are supported by at least one sub-Validator.
func Any(vs ...Validator) Validator {
	// Short-circuit trivial cases: no validators can't succeed, 1 validator can just be passed through.
	if len(vs) == 0 {
		return &noValidator{}
	} else if len(vs) == 1 {
		return vs[0]
	}
	return &anyOf{vs: vs}
}

type noValidator struct{}

func (n *noValidator) Validate(context.Context, asn1.ObjectIdentifier, []byte, []byte) error {
	return ErrOIDNotSupported
}

func (n *noValidator) String() string {
	return "<no validator>"
}

type withFixedOID struct {
	oid asn1.ObjectIdentifier
	v   Validator
}

// WithFixedOID wraps the given Validator, passing a fixed OID instead of the original one.
//
// This is only useful for backwards compatibility in the HTTP API client.
// TODO(burgerdev): remove once all clients receive the OID from the HTTP API.
func WithFixedOID(oid asn1.ObjectIdentifier, v Validator) Validator {
	// We're explicitly hiding the fact that we're using an adaptor and just use the name of the
	// wrapped validator.
	return &withFixedOID{oid: oid, v: v}
}

func (w *withFixedOID) Validate(ctx context.Context, _ asn1.ObjectIdentifier, attDoc []byte, reportData []byte) error {
	return w.v.Validate(ctx, w.oid, attDoc, reportData)
}

func (w *withFixedOID) String() string {
	return w.v.String()
}

type named struct {
	v    Validator
	name string
}

func (n *named) Validate(ctx context.Context, oid asn1.ObjectIdentifier, attDoc []byte, reportData []byte) error {
	return n.v.Validate(ctx, oid, attDoc, reportData)
}

func (n *named) String() string {
	return n.name
}

// Named wraps the given Validator and overrides fmt.Stringer with a static string.
func Named(name string, v Validator) Validator {
	return &named{v: v, name: name}
}
