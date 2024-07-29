// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/edgelesssys/contrast/node-installer/platforms"
)

// Default returns a default manifest with reference values for the given platform.
func Default(platform platforms.Platform) (*Manifest, error) {
	refValues := setReferenceValuesIfUninitialized()

	mnfst := Manifest{}
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		return &Manifest{
			ReferenceValues: ReferenceValues{
				AKS: refValues.AKS,
			},
		}, nil
	case platforms.K3sQEMUSNP:
		return &Manifest{
			ReferenceValues: ReferenceValues{
				BareMetalSNP: refValues.BareMetalSNP,
			},
		}, nil
	case platforms.RKE2QEMUTDX, platforms.K3sQEMUTDX:
		return &Manifest{
			ReferenceValues: ReferenceValues{
				BareMetalTDX: refValues.BareMetalTDX,
			},
		}, nil
	}
	return &mnfst, nil
}

// DefaultPlatformHandler is a short-hand for getting the default runtime handler for a platform.
func DefaultPlatformHandler(platform platforms.Platform) (string, error) {
	mnf, err := Default(platform)
	if err != nil {
		return "", fmt.Errorf("generating manifest: %w", err)
	}
	return mnf.RuntimeHandler(platform)
}

// EmbeddedReferenceValues returns the reference values embedded in the binary.
func EmbeddedReferenceValues() ReferenceValues {
	return setReferenceValuesIfUninitialized()
}

// EmbeddedReferenceValuesIfUninitialized returns the reference values embedded in the binary, initializing
// the global state if it is not yet initialized.
func setReferenceValuesIfUninitialized() ReferenceValues {
	if embeddedReferenceValues == nil {
		// If we're here, this is the first time this function is called, and the global state is not
		// yet initialized. So let's unmarshal the embedded reference values. This does not need to be
		// locked, as embeddedReferenceValues has a fixed value at compile time, making this idempotent.
		if err := json.Unmarshal(EmbeddedReferenceValuesJSON, &embeddedReferenceValues); err != nil {
			fmt.Printf("Failed to unmarshal embedded reference values: %s\n", err)
			os.Exit(1)
		}
	}
	return *embeddedReferenceValues
}
