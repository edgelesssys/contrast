// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package platforms

import (
	"fmt"
	"strings"
)

// RuntimeHandler returns the name of the runtime handler for the given platform.
func RuntimeHandler(platform Platform) (string, error) {
	platformName := strings.ToLower(platform.String())

	if strings.EqualFold(platformName, Unknown.String()) {
		return "", fmt.Errorf("unsupported platform %s", platform)
	}

	// TODO add hash
	hash := ""

	return fmt.Sprintf("contrast-cc-%s-%s", hash, platformName), nil
}
