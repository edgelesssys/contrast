// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package az

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// NodeImageVersion gets the node image version from the specified cluster
// and resource group.
func NodeImageVersion(ctx context.Context, clusterName string, rg string) (string, error) {
	cmd := exec.CommandContext(ctx, "az", "aks", "nodepool", "list", "--cluster-name", clusterName, "--resource-group", rg)
	stdout, err := cmd.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return "", fmt.Errorf("failed to execute az, stderr is: %s", string(exitErr.Stderr))
	} else if err != nil {
		return "", fmt.Errorf("failed to execute az: %w", err)
	}

	var outMap []map[string]any
	if err := json.Unmarshal(stdout, &outMap); err != nil {
		return "", err
	}

	if len(outMap) == 0 {
		return "", errors.New("no nodepools could be listed")
	}

	return strings.TrimSpace(fmt.Sprintf("%s", outMap[0]["nodeImageVersion"])), nil
}
