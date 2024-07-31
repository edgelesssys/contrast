// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"github.com/edgelesssys/contrast/platforms"
)

// Default returns a default manifest with reference values for the given platform.
func Default(platform platforms.Platform) (*Manifest, error) {
	refValues := platforms.EmbeddedReferenceValues()

	mnfst := Manifest{}
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		return &Manifest{
			ReferenceValues: platforms.ReferenceValues{
				AKS: refValues.AKS,
			},
		}, nil
	case platforms.RKE2QEMUTDX, platforms.K3sQEMUTDX:
		return &Manifest{
			ReferenceValues: platforms.ReferenceValues{
				BareMetalTDX: refValues.BareMetalTDX,
			},
		}, nil
	}
	return &mnfst, nil
}
