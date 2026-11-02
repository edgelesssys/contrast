// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package systemdunits

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

var (
	//go:embed expected-units.txt
	expectedUnits string
	//go:embed expected-units-gpu.txt
	expectedUnitsGPU string
)

const (
	// journalCommand lists the start jobs PID 1 completed during this boot, including those of the initrd.
	journalCommand = "journalctl --boot --no-pager --output=json --output-fields=UNIT _PID=1 JOB_TYPE=start"
	// showCommand lists all units currently loaded and if they were ever started by systemd.
	showCommand = "systemctl show --all " +
		"--property=Id,Perpetual,FragmentPath,SourcePath,ActiveEnterTimestampMonotonic,InactiveExitTimestampMonotonic '*'"
)

func parseExpected(golden string) map[string]bool {
	units := map[string]bool{}
	for line := range strings.SplitSeq(golden, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		units[fields[0]] = true
	}
	return units
}

func parseJournal(out string) ([]string, error) {
	var units []string
	for line := range strings.SplitSeq(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parsing journal entry %q: %w", line, err)
		}
		if unit, ok := entry["UNIT"].(string); ok {
			units = append(units, unit)
		}
	}
	return units, nil
}

// kernelTables are the sources of the units systemd creates for mounts and swaps it didn't set up.
var kernelTables = []string{"/proc/self/mountinfo", "/proc/swaps"}

// parseShow returns the units that have been active at some point, from the output of showCommand.
// Perpetual units are skipped, as they are always active and never started.
func parseShow(out string) (started, external []string) {
	for block := range strings.SplitSeq(out, "\n\n") {
		props := map[string]string{}
		for line := range strings.SplitSeq(block, "\n") {
			if key, value, ok := strings.Cut(line, "="); ok {
				props[key] = value
			}
		}
		if props["Id"] == "" || props["Perpetual"] == "yes" {
			continue
		}
		if props["ActiveEnterTimestampMonotonic"] == "0" && props["InactiveExitTimestampMonotonic"] == "0" {
			continue
		}
		if props["FragmentPath"] == "" && slices.Contains(kernelTables, props["SourcePath"]) {
			external = append(external, props["Id"])
		} else {
			started = append(started, props["Id"])
		}
	}
	return started, external
}

// unitTypesWithoutCode are unit types that can't execute anything.
// Devices only reflect the hardware, and scopes and slices only group processes into cgroups.
// They are skipped by the check, which also allows the transient scopes and slices that are created at runtime, e.g.
// by the Kata agent for the containers, and can't be known in advance.
var unitTypesWithoutCode = []string{".device", ".scope", ".slice"}

// checkUnits returns the observed units that aren't covered by the expected ones, and the skipped ones.
func checkUnits(observed []string, expected map[string]bool) (unexpected, skipped []string) {
	for _, unit := range observed {
		switch {
		case slices.ContainsFunc(unitTypesWithoutCode, func(s string) bool { return strings.HasSuffix(unit, s) }):
			skipped = append(skipped, unit)
		case !expected[unit] && !expected[templateOf(unit)]:
			unexpected = append(unexpected, unit)
		}
	}
	slices.Sort(unexpected)
	slices.Sort(skipped)
	return slices.Compact(unexpected), slices.Compact(skipped)
}

// templateOf turns foo@bar.service into foo@.service.
func templateOf(unit string) string {
	prefix, rest, ok := strings.Cut(unit, "@")
	dot := strings.LastIndex(rest, ".")
	if !ok || dot < 0 {
		return unit
	}
	return prefix + "@" + rest[dot:]
}
