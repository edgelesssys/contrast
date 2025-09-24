// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"github.com/edgelesssys/contrast/internal/constants"
)

// AllVerifiersBeforeGenerate returns all verifiers for k8s objects that should be run before generate.
func AllVerifiersBeforeGenerate() []Verifier {
	return []Verifier{
		// Contrast images are replaced during generate, so we can't check they are pinned
		// at this point. We run the verifier again after generate to be sure the images
		// we injected are pinned, too. We run the verifier here for all other images, to
		// give users early feedback before we pull images in generate that aren't pinned.
		&ImageRefValid{ExcludeContrastImages: true},
		&ServiceMeshEgressNotEmpty{},
	}
}

// AllVerifiersAfterGenerate returns all verifiers for k8s objects that should be run after generate.
func AllVerifiersAfterGenerate() []Verifier {
	return []Verifier{
		&NoSharedFSMount{},
		&ImageRefValid{},
		&VersionsMatch{Version: constants.Version},
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
