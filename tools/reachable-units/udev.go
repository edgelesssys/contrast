// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// udevValue matches a value of a udev rule, which is either escaped (prefixed with e) or plain (with only \" escaped),
// and may be prefixed with i for case insensitive matching, see udevValueOf and udev_rule_parse_value in systemd.
const udevValue = `((?:e|ei|ie)"(?:[^"\\]|\\.)*"|i?"(?:[^"\\]|\\"?)*")`

// reSystemdWants matches the SYSTEMD_WANTS property, which requests units when a device appears, see systemd.device(5).
var reSystemdWants = regexp.MustCompile(`ENV\{SYSTEMD_WANTS\}\s*[+:]?=\s*` + udevValue)

// rePrograms matches the programs that a rule executes, excluding builtins, see udev(7).
var rePrograms = regexp.MustCompile(`(?:RUN(?:\{program\})?|PROGRAM|IMPORT\{program\})\s*[+:=!]?=\s*` + udevValue)

// reImportFiles matches the files whose properties a rule imports, see udev(7).
var reImportFiles = regexp.MustCompile(`IMPORT\{file\}\s*[+:=!]?=\s*` + udevValue)

// reImportPrograms matches the programs whose output a rule imports as properties, see udev(7).
var reImportPrograms = regexp.MustCompile(`IMPORT\{program\}\s*[+:=!]?=\s*` + udevValue)

// reImportProperties matches the patterns of properties a rule imports from the kernel command line,
// the udev database or the parent device, see udev(7).
var reImportProperties = regexp.MustCompile(`IMPORT\{(?:cmdline|db|parent)\}\s*[+:=!]?=\s*` + udevValue)

// reHwdb matches the import of properties from the hardware database, see udev(7).
var reHwdb = regexp.MustCompile(`IMPORT\{builtin\}\s*[+:=!]?=\s*(?:e|ei|ie|i)?"hwdb\b`)

// udevValueOf returns the value of a quoted value matched by udevValue, decoding the C-style escapes of e-prefixed values.
func udevValueOf(quoted string) (string, error) {
	start := strings.Index(quoted, `"`)
	prefix, value := quoted[:start], quoted[start:]
	if strings.Contains(prefix, "e") {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("can't decode value %s: %w", quoted, err)
		}
		return unquoted, nil
	}
	return strings.ReplaceAll(value[1:len(value)-1], `\"`, `"`), nil
}

// reUnitName matches valid unit names, which systemd doesn't mangle, see unit_name_mangle in systemd.
var reUnitName = regexp.MustCompile(`^[a-zA-Z0-9:_.\\-]+(?:@[a-zA-Z0-9:_.\\-]*)?\.(?:service|socket|device|mount|automount|swap|target|path|timer|slice|scope)$`)

// systemdWants returns the unit names in the value of SYSTEMD_WANTS, split like systemd does,
// i.e., at whitespace with quotes removed and backslashes retained, see device_add_udev_wants in systemd.
func systemdWants(value string) ([]string, error) {
	var units []string
	var unit strings.Builder
	inUnit := false
	var quote rune
	for _, c := range value {
		switch {
		case quote != 0 && c == quote:
			quote = 0
		case quote != 0:
			unit.WriteRune(c)
		case c == '"' || c == '\'':
			quote, inUnit = c, true
		case strings.ContainsRune(" \t\n\r", c):
			if inUnit {
				units = append(units, unit.String())
				unit.Reset()
				inUnit = false
			}
		default:
			unit.WriteRune(c)
			inUnit = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unbalanced quotes in SYSTEMD_WANTS value %q", value)
	}
	if inUnit {
		units = append(units, unit.String())
	}
	return units, nil
}

// udevValues returns the values of all matches of the given pattern, which must end with udevValue, in the rule.
func udevValues(re *regexp.Regexp, rule string) ([]string, error) {
	var values []string
	for _, m := range re.FindAllStringSubmatch(rule, -1) {
		value, err := udevValueOf(m[1])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// udevActivations returns the units that the udev rules in the given directory start,
// either by requesting them via SYSTEMD_WANTS, or by running programs that may start units, see reviewedCommands.
// Rules that may import SYSTEMD_WANTS need a review as well, as the imported properties aren't evaluated.
// libexecDir is the directory udev resolves relative program paths against.
func udevActivations(rulesDir, libexecDir string, reviewed *reviewedCommands) ([]activation, error) {
	files, err := filepath.Glob(filepath.Join(rulesDir, "*.rules"))
	if err != nil {
		return nil, err
	}
	// udev loads the hardware database from /etc/udev and its libexec directory, see hwdb(7).
	// Check the sources too, as systemd-hwdb-update.service may compile them at runtime.
	var hwdbFiles []string
	for _, dir := range []string{filepath.Dir(rulesDir), libexecDir} {
		sources, err := filepath.Glob(filepath.Join(dir, "hwdb.d/*.hwdb"))
		if err != nil {
			return nil, err
		}
		hwdbFiles = append(append(hwdbFiles, filepath.Join(dir, "hwdb.bin")), sources...)
	}
	hwdbSetsWants := slices.ContainsFunc(hwdbFiles, func(path string) bool {
		content, err := os.ReadFile(path)
		if err != nil {
			// hwdb.bin is listed even if it doesn't exist, but an existing file that can't be read may set it.
			return !errors.Is(err, fs.ErrNotExist)
		}
		return bytes.Contains(content, []byte("SYSTEMD_WANTS"))
	})
	slices.Sort(files)
	var activations []activation
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		reason := "udev rule in " + filepath.Base(file)
		for _, line := range udevRules(string(raw)) {
			wants, err := udevValues(reSystemdWants, line)
			if err != nil {
				return nil, fmt.Errorf("in %s: %w", file, err)
			}
			for _, want := range wants {
				units, err := systemdWants(want)
				if err != nil {
					return nil, fmt.Errorf("in %s: %w", file, err)
				}
				for _, unit := range units {
					if prefix, rest, ok := strings.Cut(unit, "@"); ok {
						if dot := strings.LastIndex(rest, "."); dot >= 0 && strings.ContainsAny(rest[:dot], "$%") {
							// The instance is derived from the device, so treat it as a template.
							unit = prefix + "@." + rest[dot+1:]
						}
					}
					if strings.ContainsAny(unit, "$%") {
						return nil, fmt.Errorf("can't resolve unit %q in %s", unit, file)
					}
					if !reUnitName.MatchString(unit) {
						return nil, fmt.Errorf("%q in %s isn't a valid unit name, which systemd would mangle, use the full unit name instead", unit, file)
					}
					activations = append(activations, activation{instantiate(unit), reason})
				}
			}
			programs, err := udevValues(rePrograms, line)
			if err != nil {
				return nil, fmt.Errorf("in %s: %w", file, err)
			}
			imports, err := udevValues(reImportFiles, line)
			if err != nil {
				return nil, fmt.Errorf("in %s: %w", file, err)
			}
			importPrograms, err := udevValues(reImportPrograms, line)
			if err != nil {
				return nil, fmt.Errorf("in %s: %w", file, err)
			}
			importProperties, err := udevValues(reImportProperties, line)
			if err != nil {
				return nil, fmt.Errorf("in %s: %w", file, err)
			}
			// The whole rule is the key of the reviewed command, as it determines when the programs run.
			units, err := reviewed.lookup(line, func() bool {
				if reStartsUnits.MatchString(line) {
					return true
				}
				for _, program := range programs {
					if mayStartUnits(program, func(program string) string { return udevExecutable(program, libexecDir) }) {
						return true
					}
				}
				if slices.ContainsFunc(imports, importsSystemdWants) {
					return true
				}
				for _, patterns := range importProperties {
					// Properties of the kernel command line aren't known, and those of the database may persist from the initrd.
					// Alternatives separated by | are split like in udev matches, to stay on the safe side.
					// udev substitutes variables in the pattern first, so a pattern with substitutions may match anything.
					if strings.ContainsAny(patterns, "$%") {
						return true
					}
					for pattern := range strings.SplitSeq(patterns, "|") {
						if matches, err := path.Match(pattern, "SYSTEMD_WANTS"); err != nil || matches {
							return true
						}
					}
				}
				if hwdbSetsWants && reHwdb.MatchString(line) {
					return true
				}
				for _, program := range importPrograms {
					// The program prints the properties, so check its executable like an imported file,
					// as well as the command itself and scripts passed to it, e.g. for `sh -c 'echo SYSTEMD_WANTS=…'`.
					if strings.Contains(program, "SYSTEMD_WANTS") {
						return true
					}
					argv := strings.Fields(program)
					if len(argv) == 0 {
						continue
					}
					if importsSystemdWants(udevExecutable(argv[0], libexecDir)) || slices.ContainsFunc(argv[1:], scriptImportsSystemdWants) {
						return true
					}
				}
				return false
			})
			if err != nil {
				return nil, fmt.Errorf("in %s: %w", file, err)
			}
			for _, unit := range units {
				activations = append(activations, activation{unit, reason})
			}
		}
	}
	return activations, nil
}

// udevRules splits the content of a rules file into its trimmed rules like udev,
// skipping comment lines and then joining lines that end with a backslash.
func udevRules(content string) []string {
	var rules []string
	var rule strings.Builder
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		if continued, ok := strings.CutSuffix(line, `\`); ok {
			rule.WriteString(continued)
			continue
		}
		rule.WriteString(line)
		rules = append(rules, rule.String())
		rule.Reset()
	}
	if rule.Len() > 0 {
		rules = append(rules, rule.String())
	}
	return rules
}

// udevExecutable returns the path of the executable udev runs for a program,
// resolving relative paths against its libexec directory, see udev_event_spawn in systemd.
func udevExecutable(program, libexecDir string) string {
	if filepath.IsAbs(program) {
		return program
	}
	return filepath.Join(libexecDir, program)
}

// importsSystemdWants reports whether importing the properties of the file, or of the output of
// the executable at path, may set SYSTEMD_WANTS. As the file may only be created at runtime,
// e.g. in /run, a missing file may set it as well.
func importsSystemdWants(path string) bool {
	content, err := os.ReadFile(path)
	return err != nil || bytes.Contains(content, []byte("SYSTEMD_WANTS"))
}

// scriptImportsSystemdWants reports whether an argument of an imported program is a script that may print SYSTEMD_WANTS,
// e.g. for `sh /path/to/script`. Only absolute paths to regular files are inspected, like in mayStartUnits.
func scriptImportsSystemdWants(arg string) bool {
	if !filepath.IsAbs(arg) {
		return false
	}
	if stat, err := os.Stat(arg); err != nil || !stat.Mode().IsRegular() {
		return false
	}
	return importsSystemdWants(arg)
}
