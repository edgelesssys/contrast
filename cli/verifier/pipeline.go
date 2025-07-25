// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// VerificationPipeline represents a pipeline for resource verification.
type VerificationPipeline struct {
	Pipeline []Verifier
}

// Verify runs a given kubernetes object through the pipeline. If a verifier fails it returns an error.
func (v *VerificationPipeline) Verify(toVerify *unstructured.Unstructured) error {
	for _, toRun := range v.Pipeline {
		if err := toRun(toVerify); err != nil {
			return err
		}
	}

	return nil
}
