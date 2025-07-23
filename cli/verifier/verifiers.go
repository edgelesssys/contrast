// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// AllVerifiers returns all verifiers for k8s objects.
func AllVerifiers() []Verifier {
	return []Verifier{}
}

// VerificationFunc is a function that verifies a given unstructured object and returns an error if verification fails.
type VerificationFunc func(toVerify *unstructured.Unstructured) error

// Verify verifies a given k8s object.
func (f VerificationFunc) Verify(toVerify *unstructured.Unstructured) error {
	return f(toVerify)
}

// Verifier verifies a given k8s object.
type Verifier interface {
	Verify(toVerify *unstructured.Unstructured) error
}
