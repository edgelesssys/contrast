// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package az

import (
	"os/exec"
	"testing"
)

// KataPolicyGen executes `az confcom katapolicygen --yaml <resourcePath>`.
func KataPolicyGen(t *testing.T, resourcePath string) error {
	// log versions and extensions that are used
	out, err := exec.Command("az", "confcom", "katapolicygen", "--print-version").Output()
	if err != nil {
		return err
	}
	t.Log(string(out))

	return exec.Command("az", "confcom", "katapolicygen", "--yaml", resourcePath).Run()
}
