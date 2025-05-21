// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package az

import (
	"os"
	"os/exec"
	"strings"
)

// KataPolicyGenVersion gets the version string of `az confcom katapolicygen`.
func KataPolicyGenVersion() (string, error) {
	out, err := exec.Command("az", "confcom", "katapolicygen", "--print-version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// KataPolicyGen executes `az confcom katapolicygen --yaml <resourcePath>`.
func KataPolicyGen(resourcePath string) error {
	cmd := exec.Command("az", "confcom", "katapolicygen", "--yaml", resourcePath)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
