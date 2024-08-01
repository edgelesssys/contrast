// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"encoding/json"
	"fmt"

	"github.com/edgelesssys/contrast/internal/platforms"
)

// RuntimeHandler returns the name of the runtime handler for the given platform.
func RuntimeHandler(platform platforms.Platform) (string, error) {
	var mapping EmbeddedReferenceValues
	if err := json.Unmarshal(EmbeddedReferenceValuesJSON, &mapping); err != nil {
		return "", fmt.Errorf("unmarshal embedded reference values mapping: %w", err)
	}

	for runtimeHandler := range mapping {
		p, err := platformFromHandler(runtimeHandler)
		if err != nil {
			return "", fmt.Errorf("invalid runtime handler name %s: %w", runtimeHandler, err)
		}

		if p == platform {
			return runtimeHandler, nil
		}
	}

	return "", fmt.Errorf("no runtime handler found for platform %s", platform)
}
