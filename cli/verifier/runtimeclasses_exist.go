// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"
	"fmt"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/spf13/cobra"

	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// RuntimeClassesExist verifies that all used contrast-cc or contrast-insecure prefixed runtimeClassNames are valid.
type RuntimeClassesExist struct {
	Command *cobra.Command
}

// Verify verifies that all used contrast-cc or contrast-insecure prefixed runtimeClassNames are valid.
func (r *RuntimeClassesExist) Verify(toVerify any) error {
	var collectedErrs error
	collectedMissingRuntimes := map[string]error{}

	defaultRuntimeClass, err := r.Command.Flags().GetString("reference-values")
	if err != nil {
		return err
	}

	kuberesource.MapPodSpec(toVerify, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		if !kuberesource.IsContrastPod(spec) {
			return spec
		}
		// Bare runtime class names (without hash suffix) are placeholders that
		// get resolved during generate. They can't be parsed as platforms.
		if *spec.RuntimeClassName == "contrast-cc" || *spec.RuntimeClassName == "contrast-insecure" {
			if defaultRuntimeClass == "" {
				collectedMissingRuntimes[*spec.RuntimeClassName] = fmt.Errorf("no default platform was specified using --reference-values")
			}
			return spec
		}

		_, err := platforms.FromRuntimeClassString(*spec.RuntimeClassName)
		if err != nil {
			// This swallows all but the latest error for a given runtime class name.
			// However, it's unlikely that multiple error sources occur for the same
			// runtimeClassName, and this prevents duplicate error messages.
			collectedMissingRuntimes[*spec.RuntimeClassName] = err
		}
		return spec
	})

	for name, err := range collectedMissingRuntimes {
		collectedErrs = errors.Join(collectedErrs, fmt.Errorf("%q is not a valid runtime class name: %w", name, err))
	}

	return collectedErrs
}
