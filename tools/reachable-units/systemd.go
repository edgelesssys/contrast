// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// stage runs the service manager of a boot stage in test mode.
type stage struct {
	cfg config
	env []string
}

func newStage(cfg config) *stage {
	inInitrd := "0"
	if cfg.InInitrd {
		inInitrd = "1"
	}
	env := append(os.Environ(),
		"SYSTEMD_IN_INITRD="+inInitrd,
		"SYSTEMD_GENERATOR_PATH="+strings.Join(cfg.GeneratorDirs, ":"),
	)
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}
	return &stage{cfg: cfg, env: env}
}

// transaction returns the dump of `systemd --test` for starting the given unit.
func (s *stage) transaction(ctx context.Context, unit string) (*dump, error) {
	out, _, err := s.test(ctx, unit, "warning")
	if err != nil {
		return nil, err
	}
	return parseDump(out), nil
}

// generators returns the generators that systemd runs, including environment generators.
// Generators that aren't needed should be masked in the image.
func (s *stage) generators(ctx context.Context) ([]string, error) {
	_, log, err := s.test(ctx, s.cfg.Target, "debug")
	if err != nil {
		return nil, err
	}
	return parseGenerators(log)
}

// parseGenerators returns the names of the generators that were executed according to the debug log of systemd, or an error if one of them didn't succeed.
func parseGenerators(log string) ([]string, error) {
	succeeded := map[string]bool{}
	for _, m := range reSucceeded.FindAllStringSubmatch(log, -1) {
		succeeded[m[1]] = true
	}
	var generators []string
	for _, m := range reExecuted.FindAllStringSubmatch(log, -1) {
		if !succeeded[m[1]] {
			return nil, fmt.Errorf("generator %s didn't succeed:\n%s", m[1], log)
		}
		generators = append(generators, filepath.Base(m[1]))
	}
	slices.Sort(generators)
	return generators, nil
}

var (
	reExecuted  = regexp.MustCompile(`(?m)^About to execute (\S+)$`)
	reSucceeded = regexp.MustCompile(`(?m)^'(\S+)' succeeded\.$`)
)

// test runs `systemd --test` for the given unit and returns its stdout and stderr.
func (s *stage) test(ctx context.Context, unit, logLevel string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, filepath.Join(s.cfg.Systemd, "lib/systemd/systemd"),
		"--test", "--system", "--no-pager", "--unit="+unit)
	cmd.Env = append(slices.Clone(s.env), "SYSTEMD_LOG_LEVEL="+logLevel)
	var errBuf strings.Builder
	cmd.Stderr = &errBuf
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("systemd --test --unit=%s: %w\n%s", unit, err, errBuf.String())
	}
	return string(out), errBuf.String(), nil
}
