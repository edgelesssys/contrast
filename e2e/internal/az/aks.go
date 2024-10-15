// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package az

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// NodeImageVersion gets the node image version from the specified cluster
// and resource group.
func NodeImageVersion(clusterName string, rg string) (string, error) {
	out, err := exec.Command("az", "aks", "nodepool", "list", "--cluster-name", clusterName, "--resource-group", rg).Output()
	if err != nil {
		return "", err
	}

	var outMap []map[string]interface{}
	err = json.Unmarshal(out, &outMap)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s", outMap[0]["nodeImageVersion"]), nil
}