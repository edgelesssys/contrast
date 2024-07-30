// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package runtimehandler

import (
	"fmt"
	"strings"

	"github.com/edgelesssys/contrast/node-installer/internal/constants"
	"github.com/edgelesssys/contrast/node-installer/platforms"
)

// Name returns the name of the runtime handler for the given platform.
func Name(platform platforms.Platform) (string, error) {
	platformName := strings.ToLower(platform.String())

	if strings.EqualFold(platformName, platforms.Unknown.String()) {
		return "", fmt.Errorf("unsupported platform %s", platform)
	}

	// Replace dots to ensure a readable directory name used by the node-installer.
	version := strings.ReplaceAll(constants.Version, ".", "-")

	return fmt.Sprintf("contrast-cc-%s-%s", version, platformName), nil
}
