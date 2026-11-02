// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// reviewedCommands are the command lines that may start units, see mayStartUnits, mapped to the units they start.
type reviewedCommands struct {
	starts map[string][]string
	// used maps the reviewed command lines that were encountered to the full command line, including store paths.
	used map[string]string
}

func newReviewedCommands(starts map[string][]string) *reviewedCommands {
	return &reviewedCommands{starts: starts, used: map[string]string{}}
}

// unitsStartedBy returns the units that a command line starts.
func (r *reviewedCommands) unitsStartedBy(commandLine string) ([]string, error) {
	return r.lookup(commandLine, func() bool {
		return mayStartUnits(commandLine, func(program string) string {
			// systemd looks up programs without a path at runtime, in a search path that depends on the environment
			// of the unit, which isn't evaluated here. Such programs aren't inspected, only their command line is.
			if !strings.Contains(program, "/") {
				return ""
			}
			return program
		})
	})
}

// lookup returns the units that a command line starts, according to its review.
// Command lines without a review must not start units according to mayStart, which is only called for those.
func (r *reviewedCommands) lookup(commandLine string, mayStart func() bool) ([]string, error) {
	key := withoutStorePaths(commandLine)
	if units, ok := r.starts[key]; ok {
		// Without store paths, programs of different packages may share the reviewed command line.
		if prev, ok := r.used[key]; ok && prev != commandLine {
			return nil, fmt.Errorf("%q is ambiguous, it matches both %q and %q", key, prev, commandLine)
		}
		r.used[key] = commandLine
		return units, nil
	}
	if mayStart() {
		return nil, fmt.Errorf("%q may start units, add it to the reviewed commands along with the units it starts", key)
	}
	return nil, nil
}

// unused returns the reviewed command lines that weren't encountered, so they can be removed.
func (r *reviewedCommands) unused() []string {
	var unused []string
	for _, key := range slices.Sorted(maps.Keys(r.starts)) {
		if _, ok := r.used[key]; !ok {
			unused = append(unused, key)
		}
	}
	return unused
}

// reStartsUnits matches the ways a command can start units: systemctl, systemd-run (which can pull in units via
// dependencies of the transient unit), and D-Bus calls to the service manager, e.g. with busctl or dbus-send.
var reStartsUnits = regexp.MustCompile(`systemctl|systemd-run|org\.freedesktop\.systemd1`)

// mayStartUnits reports whether the command line, or a script it executes, may start units, see reStartsUnits.
// resolve returns the path of the executable for the program in the command line, or "" if it can't be resolved.
//
// A resolved executable must be inspectable, as it may start units otherwise, e.g. if it is only created at runtime in /run.
// Scripts passed to an interpreter as an absolute path are inspected too, e.g. for `sh /path/to/script`.
// Binaries can't be inspected, so they are assumed not to start units.
func mayStartUnits(commandLine string, resolve func(program string) string) bool {
	if reStartsUnits.MatchString(commandLine) {
		return true
	}
	argv := strings.Fields(commandLine)
	if len(argv) == 0 {
		return false
	}
	if executable := resolve(argv[0]); executable != "" {
		if startsUnits, ok := scriptStartsUnits(executable); !ok || startsUnits {
			return true
		}
	}
	for _, arg := range argv[1:] {
		if !filepath.IsAbs(arg) {
			continue
		}
		if startsUnits, _ := scriptStartsUnits(arg); startsUnits {
			return true
		}
	}
	return false
}

// reScriptRef matches the store paths of files that a script references, e.g. the script a wrapper executes.
var reScriptRef = regexp.MustCompile("/nix/store/[0-9a-z]{32}-[^\\s\"'`;|&<>(){}$\\\\]+")

// scriptStartsUnits reports whether the file at the path is a script that may start units, see reStartsUnits,
// or references such a script by its store path, e.g. as a wrapper does.
// ok is false if the path isn't a regular file that can be read.
func scriptStartsUnits(path string) (startsUnits, ok bool) {
	return inspectScript(path, map[string]bool{})
}

func inspectScript(path string, seen map[string]bool) (startsUnits, ok bool) {
	if seen[path] {
		return false, true
	}
	seen[path] = true
	// Only read regular files, as reading a device such as /dev/zero may not terminate.
	if stat, err := os.Stat(path); err != nil || !stat.Mode().IsRegular() {
		return false, false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return false, false
	}
	if bytes.HasPrefix(content, []byte("\x7fELF")) {
		return false, true
	}
	if reStartsUnits.Match(content) {
		return true, true
	}
	for _, ref := range reScriptRef.FindAll(content, -1) {
		if startsUnits, _ := inspectScript(string(ref), seen); startsUnits {
			return true, true
		}
	}
	return false, true
}

var (
	reStorePath       = regexp.MustCompile(`/nix/store/[0-9a-z]{32}-[^/\s]+/(?:s?bin/)?`)
	reStoreFilePrefix = regexp.MustCompile(`/nix/store/[0-9a-z]{32}-`)
)

func withoutStorePaths(commandLine string) string {
	return reStoreFilePrefix.ReplaceAllString(reStorePath.ReplaceAllString(commandLine, ""), "")
}
