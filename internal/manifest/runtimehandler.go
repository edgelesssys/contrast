// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/edgelesssys/contrast/internal/platforms"
)

// RuntimeHandler returns the name of the runtime handler for the given platform.
func RuntimeHandler(platform platforms.Platform) (string, error) {
	var mapping PlatformRuntimeMapping
	if err := json.Unmarshal(EmbeddedPlatformRuntimeMappingJSON, &mapping); err != nil {
		return "", fmt.Errorf("unmarshal embedded platform handler mapping: %w", err)
	}

	var runtimeHash string
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		runtimeHash = mapping.AKS
	case platforms.RKE2QEMUTDX, platforms.K3sQEMUTDX:
		runtimeHash = mapping.BareMetalTDX
	case platforms.K3sQEMUSNP:
		runtimeHash = mapping.BareMetalSNP
	default:
		return "", fmt.Errorf("unsupported platform %s", platform)
	}

	return fmt.Sprintf("contrast-cc-%s-%s", strings.ToLower(platform.String()), runtimeHash[:8]), nil
}
