// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"encoding/json"
	"fmt"

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

// EmbeddedReferenceValuesIfUninitialized returns the reference values embedded in the binary.
func setReferenceValuesIfUninitialized() ReferenceValues {
	var embeddedReferenceValues *ReferenceValues

	if err := json.Unmarshal(EmbeddedReferenceValuesJSON, &embeddedReferenceValues); err != nil {
		// As this relies on a constant, predictable value (i.e. the embedded JSON), which -- in a correctly built binary -- should
		// unmarshal safely into the [ReferenceValues], it's acceptable to panic here.
		panic(fmt.Errorf("failed to unmarshal embedded reference values: %w", err))
	}

	return *embeddedReferenceValues
}
