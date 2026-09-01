// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apitypes

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseAPIVersion parses an API version identifier like "v1" into its numeric version.
func ParseAPIVersion(version string) (int, error) {
	digits, ok := strings.CutPrefix(version, "v")
	if !ok || digits == "" || digits[0] == '0' || strings.TrimLeft(digits, "0123456789") != "" {
		return 0, fmt.Errorf("API version %q: expected \"v\" followed by a positive integer", version)
	}
	n, err := strconv.Atoi(digits)
	if err != nil {
		return 0, fmt.Errorf("API version %q: %w", version, err)
	}
	return n, nil
}
