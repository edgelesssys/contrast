// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Verifier is a function that verifies a given unstructured object and returns an error if verification fails.
type Verifier func(toVerify *unstructured.Unstructured) error
