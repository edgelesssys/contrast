// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// dbusActivations returns the units that the bus daemon starts when a client accesses one of the D-Bus activatable services,
// which are described by the service files in the given directories.
func dbusActivations(serviceDirs []string) ([]activation, error) {
	var activations []activation
	seen := map[string]bool{}
	for _, dir := range serviceDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.service"))
		if err != nil {
			return nil, err
		}
		slices.Sort(files)
		for _, file := range files {
			raw, err := os.ReadFile(file)
			if err != nil {
				return nil, err
			}
			var name, unit string
			for line := range strings.SplitSeq(string(raw), "\n") {
				key, value, _ := strings.Cut(line, "=")
				key, value = strings.TrimSpace(key), strings.TrimSpace(value)
				switch key {
				case "Name":
					name = value
				case "SystemdService":
					unit = value
				}
			}
			if name == "org.freedesktop.systemd1" {
				continue // Served by PID 1 itself.
			}
			if seen[name] {
				continue // The bus daemon uses the definition from the earliest directory.
			}
			seen[name] = true
			if unit == "" {
				// The bus daemon would spawn the service itself, outside of any unit.
				return nil, fmt.Errorf("D-Bus service %s has no SystemdService=", file)
			}
			activations = append(activations, activation{unit, "D-Bus activation of " + name})
		}
	}
	return activations, nil
}
