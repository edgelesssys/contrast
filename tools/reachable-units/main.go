// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// reachable-units lists every systemd unit that can eventually be started in a boot stage of an image.
//
// Starting from the target of the stage, it follows every way in which a unit can cause further units to be started:
//
//   - the transaction for starting a unit, which pulls in its dependencies,
//   - socket, timer and path units triggering their service,
//   - sockets with Accept=yes, spawning an instance of their service template per connection,
//   - OnFailure=, OnSuccess= and Upholds= dependencies,
//   - emergency actions such as FailureAction=poweroff, starting the respective target,
//   - commands in Exec*= settings that may start units, e.g. via systemctl (these need to be manually reviewed by us!),
//   - D-Bus activation, once a bus daemon runs,
//   - udev rules requesting units via SYSTEMD_WANTS, or running such commands, once udevd runs,
//   - targets the service manager starts when the kernel signals it, e.g. on Ctrl-Alt-Del.
//
// Hardware is considered regardless of whether it is present, so the result is an upper bound.
//
// Usage: reachable-units <stage.json>

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"os/signal"
	"slices"
	"strings"
)

// config describes a boot stage of an image, see packages/by-name/kata/systemd-units.
type config struct {
	// Name of the stage, used in the output.
	Name string `json:"name"`
	// InInitrd tells whether the stage runs in the initrd.
	InInitrd bool `json:"inInitrd"`
	// Target is the unit started on boot.
	Target string `json:"target"`
	// Systemd is the store path of the systemd package used by the stage.
	Systemd string `json:"systemd"`
	// Etc is the /etc directory of the stage, which holds the units of the image.
	Etc string `json:"etc"`
	// GeneratorDirs are the directories generators are loaded from, in order of priority.
	GeneratorDirs []string `json:"generatorDirs"`
	// DBusServiceDirs are the directories of the system bus holding the service files.
	DBusServiceDirs []string `json:"dbusServiceDirs"`
	// ReviewedCommands are the command lines that may start units, without store paths, mapped to the units they start, see reviewedCommands.
	ReviewedCommands map[string][]string `json:"reviewedCommands"`
	// Env are environment variables passed to systemd, which also passes them to the generators, e.g. to set the kernel command line through SYSTEMD_PROC_CMDLINE.
	Env map[string]string `json:"env"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: reachable-units <stage.json>")
		os.Exit(2)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configPath string) error {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("parsing %s: %w", configPath, err)
	}
	if err := os.Symlink(cfg.Etc, "/etc"); err != nil {
		return fmt.Errorf("providing /etc of the stage, is the tool running isolated?: %w", err)
	}
	s := newStage(cfg)
	generators, err := s.generators(ctx)
	if err != nil {
		return err
	}
	reached, err := newAnalysis(cfg, s.transaction).run(ctx)
	if err != nil {
		return err
	}
	return write(os.Stdout, cfg, generators, reached)
}

func write(w io.Writer, cfg config, generators []string, reached map[string]string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s (%s)\n#\n# generators:\n", cfg.Name, cfg.Target)
	for _, gen := range generators {
		fmt.Fprintf(&b, "#   %s\n", gen)
	}
	fmt.Fprintln(&b, "#")

	units := slices.Collect(maps.Keys(reached))
	slices.SortFunc(units, func(a, b string) int { return strings.Compare(displayName(a), displayName(b)) })
	for _, unit := range units {
		if reached[unit] == reasonBoot {
			fmt.Fprintln(&b, displayName(unit))
		} else {
			fmt.Fprintf(&b, "%s (%s)\n", displayName(unit), reached[unit])
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
