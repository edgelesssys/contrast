// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/spf13/cobra"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// AllVerifiersBeforeGenerate returns all verifiers for k8s objects that should be run before generate.
func AllVerifiersBeforeGenerate(cmd *cobra.Command) []Verifier {
	return []Verifier{
		// Contrast images are replaced during generate, so we can't check they are pinned
		// at this point. We run the verifier again after generate to be sure the images
		// we injected are pinned, too. We run the verifier here for all other images, to
		// give users early feedback before we pull images in generate that aren't pinned.
		&ImageRefValid{ExcludeContrastImages: true},
		&ServiceMeshEgressNotEmpty{},
		&RuntimeClassesExist{Command: cmd},
		ContrastPodSpecVerifier(CPUCountValid),
		ContrastPodSpecVerifier(NoHostNetwork),
	}
}

// AllVerifiersAfterGenerate returns all verifiers for k8s objects that should be run after generate.
func AllVerifiersAfterGenerate() []Verifier {
	return []Verifier{
		ContrastPodSpecVerifier(NoSharedFSMount),
		&ImageRefValid{},
		&VersionsMatch{Version: constants.Version},
	}
}

// Verifier verifies a given k8s object. `toVerify` should be an apply configuration.
type Verifier interface {
	// Verify is called by the CLI, passing in an applyconfiguration.
	//
	// Implementations don't need to include resource identifiers in the error message.
	Verify(toVerify any) error
}

// ContrastPodSpecVerifier is a simple verification function without side-effects.
//
// The function is only applied to Contrast pods, other pods are assumed to pass.
type ContrastPodSpecVerifier func(spec *applycorev1.PodSpecApplyConfiguration) error

// Verify implements Verifier for ContrastPodSpecVerifier.
func (f ContrastPodSpecVerifier) Verify(toVerify any) error {
	_, err := kuberesource.MapPodSpecWithErrors(toVerify, func(spec *applycorev1.PodSpecApplyConfiguration) (*applycorev1.PodSpecApplyConfiguration, error) {
		if spec == nil || !kuberesource.IsContrastPod(spec) {
			return spec, nil
		}
		return spec, f(spec)
	})
	return err
}
