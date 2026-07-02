// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package insecure

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

// kernelCmdlinePath is the path to the kernel command line.
const kernelCmdlinePath = "/proc/cmdline"

// cmdlineFlag is the kernel command-line flag that opts a node into insecure (non-CC) attestation.
const cmdlineFlag = "contrast.allow_insecure_attestation=1"

// AttestationAllowed reports whether the kernel command line opts this node into insecure (non-CC)
// attestation via the contrast.allow_insecure_attestation=1 flag.
func AttestationAllowed() (bool, error) {
	cmdline, err := os.ReadFile(kernelCmdlinePath)
	if err != nil {
		return false, fmt.Errorf("reading kernel command line: %w", err)
	}
	return slices.Contains(strings.Fields(string(cmdline)), cmdlineFlag), nil
}
