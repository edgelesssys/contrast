// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package upstream

import (
	"fmt"

	"github.com/edgelesssys/contrast/internal/platforms"
)

// BaseConfig returns the embedded upstream Kata runtime configuration for the given platform.
func BaseConfig(platform platforms.Platform) ([]byte, error) {
	switch {
	case platforms.IsTDX(platform):
		return []byte(kataBareMetalQEMUTDXBaseConfig), nil
	case platforms.IsSNP(platform) || platforms.IsInsecure(platform):
		return []byte(kataBareMetalQEMUSNPBaseConfig), nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}
