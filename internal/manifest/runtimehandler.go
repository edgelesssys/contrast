// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"fmt"
	"strings"

	"github.com/edgelesssys/contrast/internal/platforms"
)

// RuntimeHandler returns the name of the runtime handler for the given platform.
func RuntimeHandler(platform platforms.Platform) (string, error) {
	mapping, err := GetEmbeddedReferenceValues()
	if err != nil {
		return "", err
	}
	for runtimeHandler := range mapping {
		p, err := PlatformFromHandler(runtimeHandler)
		if err != nil {
			return "", fmt.Errorf("invalid runtime handler name %s: %w", runtimeHandler, err)
		}

		if p == platform {
			return runtimeHandler, nil
		}
	}

	return "", fmt.Errorf("no runtime handler found for platform %s", platform)
}

// PlatformFromHandler extracts the platform from the runtime handler name.
func PlatformFromHandler(handler string) (platforms.Platform, error) {
	rest, found := strings.CutPrefix(handler, "contrast-cc-")
	isInsecure := false
	if !found {
		rest, found = strings.CutPrefix(handler, "contrast-insecure-")
		isInsecure = true
	}
	if !found {
		return platforms.Unknown, fmt.Errorf("invalid handler name: %s", handler)
	}

	parts := strings.Split(rest, "-")
	if len(parts) != 4 && len(parts) != 5 {
		return platforms.Unknown, fmt.Errorf("invalid handler name: %s", handler)
	}

	rawPlatform := strings.Join(parts[:len(parts)-1], "-")
	if isInsecure {
		rawPlatform += "-insecure"
	}

	platform, err := platforms.FromString(rawPlatform)
	if err != nil {
		return platforms.Unknown, fmt.Errorf("invalid platform in handler name: %w", err)
	}

	return platform, nil
}
