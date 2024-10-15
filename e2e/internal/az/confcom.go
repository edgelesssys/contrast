// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package az

import (
	"os/exec"
	"strings"
	"testing"
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
func KataPolicyGen(t *testing.T, resourcePath string) error {
	return exec.Command("az", "confcom", "katapolicygen", "--yaml", resourcePath).Run()
}
