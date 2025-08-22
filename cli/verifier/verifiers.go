// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

// AllVerifiers returns all verifiers for k8s objects.
func AllVerifiers() []Verifier {
	return []Verifier{
		&NoSharedFSMount{},
	}
}

// VerificationFunc is a function that verifies a given apply configuration and returns an error if verification fails.
type VerificationFunc func(toVerify any) error

// Verify verifies a given k8s object. `toVerify` should be an apply configuration.
func (f VerificationFunc) Verify(toVerify any) error {
	return f(toVerify)
}

// Verifier verifies a given k8s object. `toVerify` should be an apply configuration.
type Verifier interface {
	Verify(toVerify any) error
}
