// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"regexp"
	"strings"
)

// unitInfo is what we take from the dump of a single unit.
type unitInfo struct {
	loadState   string
	activeState string
	perpetual   bool
	// fragmentPath is the unit file the unit is loaded from, if any.
	fragmentPath string
	// accept is set for sockets with Accept=yes.
	accept bool
	// deps maps a dependency type from activatingDeps to the units it names.
	deps map[string][]string
	// actions maps an emergency action setting from emergencyActions to its value.
	actions map[string]string
	// commands holds the command lines of all Exec*= settings.
	commands []string
}

// dump is the parsed output of `systemd --test`.
type dump struct {
	// units maps the primary name of every loaded unit to what we know about it.
	units map[string]*unitInfo
	// aliases maps an alias to the primary name of its unit.
	aliases map[string]string
	// jobs are the units that get a start job in the transaction.
	jobs []string
}

var (
	reUnit        = regexp.MustCompile(`^\t→ Unit (\S+):$`)
	reField       = regexp.MustCompile(`^\t\t([A-Za-z ]+): (.*)$`)
	reCommandLine = regexp.MustCompile(`^\t\t\tCommand Line: (.*)$`)
	reJob         = regexp.MustCompile(`^\t\tAction: (\S+) -> start$`)
)

// parseDump parses the output of `systemd --test --system --unit=...`, which describes all loaded units and the jobs of the transaction for starting the given unit.
func parseDump(out string) *dump {
	d := &dump{units: map[string]*unitInfo{}, aliases: map[string]string{}}
	units, jobs, _ := strings.Cut(out, "\n-> By jobs:\n")

	var name string
	var info *unitInfo
	for line := range strings.SplitSeq(units, "\n") {
		if m := reUnit.FindStringSubmatch(line); m != nil {
			name = m[1]
			info = &unitInfo{deps: map[string][]string{}, actions: map[string]string{}}
			d.units[name] = info
			continue
		}
		if info == nil {
			continue
		}
		if m := reCommandLine.FindStringSubmatch(line); m != nil {
			info.commands = append(info.commands, m[1])
			continue
		}
		m := reField.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		switch key, value := m[1], m[2]; {
		case key == "Unit Load State":
			info.loadState = value
		case key == "Unit Active State":
			info.activeState = value
		case key == "Perpetual":
			info.perpetual = value == "yes"
		case key == "Fragment Path":
			info.fragmentPath = value
		case key == "Alias":
			d.aliases[value] = name
		case key == "Accepted": // Only printed for sockets with Accept=yes.
			info.accept = true
		case activatingDeps[key] != "":
			// The unit is followed by the origin of the dependency, e.g. "foo.service (origin-file)".
			info.deps[key] = append(info.deps[key], strings.Fields(value)[0])
		case emergencyActions[key]:
			info.actions[key] = value
		}
	}

	for line := range strings.SplitSeq(jobs, "\n") {
		if m := reJob.FindStringSubmatch(line); m != nil {
			d.jobs = append(d.jobs, m[1])
		}
	}
	return d
}
