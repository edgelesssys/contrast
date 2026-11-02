// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"
	"path/filepath"
)

// checkNoDeviceDeps checks that no device unit in the given unit directory has dependencies, i.e., units that are started when the device appears.
func checkNoDeviceDeps(unitDir string) error {
	for _, pattern := range []string{"*.device", "*.device.wants", "*.device.requires", "*.device.upholds", "*.device.d", "device.d"} {
		matches, err := filepath.Glob(filepath.Join(unitDir, pattern))
		if err != nil {
			return err
		}
		if len(matches) > 0 {
			return fmt.Errorf("dependencies of device units aren't supported, found %s", matches[0])
		}
	}
	return nil
}
