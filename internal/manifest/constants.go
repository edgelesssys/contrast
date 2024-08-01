// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"encoding/json"
	"fmt"

	"github.com/edgelesssys/contrast/internal/platforms"
)

// Default returns a default manifest with reference values for the given platform.
func Default(platform platforms.Platform) (*Manifest, error) {
	embeddedRefValues := GetEmbeddedReferenceValues()
	refValues, err := embeddedRefValues.ForPlatform(platform)
	if err != nil {
		return nil, fmt.Errorf("get reference values for platform %s: %w", platform, err)
	}

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

// GetEmbeddedReferenceValues returns the reference values embedded in the binary.
func GetEmbeddedReferenceValues() EmbeddedReferenceValues {
	var embeddedReferenceValues EmbeddedReferenceValues

	if err := json.Unmarshal(EmbeddedReferenceValuesJSON, &embeddedReferenceValues); err != nil {
		// As this relies on a constant, predictable value (i.e. the embedded JSON), which -- in a correctly built binary -- should
		// unmarshal safely into the [ReferenceValues], it's acceptable to panic here.
		panic(fmt.Errorf("failed to unmarshal embedded reference values: %w", err))
	}

	return embeddedReferenceValues
}
