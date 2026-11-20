// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// activatingDeps are the dependencies that cause the named units to be started at some point, after the depending unit has been started.
// They are mapped to the phrase explaining why.
var activatingDeps = map[string]string{
	"Triggers":  "triggered by",
	"OnFailure": "on failure of",
	"OnSuccess": "on success of",
	"Upholds":   "upheld by",
}

// emergencyActions are the settings of a unit that run an emergency action, as named in the dump.
//
// An emergency action other than none starts the target of the same name, see systemd.unit(5),
// There is no target of their name, so it's skipped by the analysis.
var emergencyActions = map[string]bool{
	"Failure Action":     true,
	"Success Action":     true,
	"Job Timeout Action": true,
	"Start Limit Action": true,
}

// unitActivations returns the units that the given unit can start once it has been started, excluding those pulled into the transaction for starting it.
func unitActivations(name string, info *unitInfo, reviewed *reviewedCommands) ([]activation, error) {
	var activations []activation
	for _, dep := range slices.Sorted(maps.Keys(info.deps)) {
		for _, unit := range info.deps[dep] {
			activations = append(activations, activation{unit, activatingDeps[dep] + " " + displayName(name)})
		}
	}

	if info.accept {
		// A socket with Accept=yes spawns an instance of the service template of the same name per connection, see systemd.socket(5).
		template := strings.TrimSuffix(name, ".socket") + "@.service"
		activations = append(activations, activation{instantiate(template), "accepted by " + name})
	}

	for _, setting := range slices.Sorted(maps.Keys(info.actions)) {
		if action := info.actions[setting]; action != "none" {
			reason := strings.ToLower(setting) + " of " + displayName(name)
			activations = append(activations, activation{action + ".target", reason})
		}
	}

	for _, command := range info.commands {
		units, err := reviewed.unitsStartedBy(command)
		if err != nil {
			return nil, fmt.Errorf("command of %s: %w", name, err)
		}
		for _, unit := range units {
			activations = append(activations, activation{unit, "started by command of " + displayName(name)})
		}
	}
	return activations, nil
}
