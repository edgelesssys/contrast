// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"maps"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSystemd mimics `systemd --test` for a fixed set of units.
type fakeSystemd struct {
	units   map[string]*unitInfo
	aliases map[string]string
	// jobs maps a unit to the jobs of the transaction for starting it. If a unit isn't listed, the transaction only contains the unit itself.
	jobs map[string][]string
	// started are the units transactions were computed for.
	started []string
}

func (f *fakeSystemd) transaction(_ context.Context, unit string) (*dump, error) {
	f.started = append(f.started, unit)
	d := &dump{units: maps.Clone(f.units), aliases: f.aliases, jobs: f.jobs[unit]}
	primary := unit
	if p, ok := f.aliases[unit]; ok {
		primary = p
	}
	if _, ok := f.units[primary]; !ok {
		d.units[primary] = &unitInfo{loadState: "not-found", activeState: "inactive"}
		return d, nil
	}
	if d.jobs == nil {
		d.jobs = []string{primary}
	}
	return d, nil
}

// loaded returns the given unit info with the states of a unit that is loaded but not started.
func loaded(info unitInfo) *unitInfo {
	info.loadState = "loaded"
	if info.activeState == "" {
		info.activeState = "inactive"
	}
	return &info
}

func TestAnalysis(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"dbus/services/org.example.service": "[D-BUS Service]\nName=org.example\n" +
			"SystemdService=dbus-org.example.service\n",
		"etc/udev/rules.d/99-test.rules": `SUBSYSTEM=="sound", ENV{SYSTEMD_WANTS}+="sound.target"`,
	})
	cfg := config{
		Target:           "boot.target",
		DBusServiceDirs:  []string{filepath.Join(dir, "dbus/services")},
		Etc:              filepath.Join(dir, "etc"),
		ReviewedCommands: map[string][]string{"systemctl --force poweroff": {}},
	}
	systemd := &fakeSystemd{
		units: map[string]*unitInfo{
			"boot.target": loaded(unitInfo{}),
			"-.slice":     loaded(unitInfo{perpetual: true, activeState: "active"}),
			"agent.service": loaded(unitInfo{
				deps:     map[string][]string{"OnFailure": {"emergency.target"}},
				actions:  map[string]string{"Failure Action": "poweroff"},
				commands: []string{"/nix/store/0123456789abcdfghijklmnpqrsvwxyz-systemd/bin/systemctl --force poweroff"},
			}),
			"poweroff.target":   loaded(unitInfo{}),
			"emergency.target":  loaded(unitInfo{}),
			"emergency.service": loaded(unitInfo{}),
			"shell.socket":      loaded(unitInfo{accept: true}),
			"shell@reachable-units-placeholder.service": loaded(unitInfo{}),
			"helper.service":        loaded(unitInfo{}),
			"dbus.socket":           loaded(unitInfo{deps: map[string][]string{"Triggers": {"dbus-broker.service"}}}),
			"dbus-broker.service":   loaded(unitInfo{}),
			"example.service":       loaded(unitInfo{}),
			"systemd-udevd.service": loaded(unitInfo{}),
			"sound.target":          loaded(unitInfo{}),
			"reboot.target":         loaded(unitInfo{}),
			"never.service":         loaded(unitInfo{}),
		},
		aliases: map[string]string{
			"dbus-org.example.service": "example.service",
			"ctrl-alt-del.target":      "reboot.target",
		},
		jobs: map[string][]string{
			"boot.target": {
				"boot.target", "agent.service", "shell.socket", "dbus.socket", "systemd-udevd.service", "dev-vda.device",
			},
			"emergency.target":                          {"emergency.target", "emergency.service"},
			"shell@reachable-units-placeholder.service": {"shell@reachable-units-placeholder.service", "helper.service"},
		},
	}

	reached, err := newAnalysis(cfg, systemd.transaction).run(t.Context())
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"boot.target":           reasonBoot,
		"agent.service":         reasonBoot,
		"shell.socket":          reasonBoot,
		"dbus.socket":           reasonBoot,
		"systemd-udevd.service": reasonBoot,
		"-.slice":               reasonPerpetual,
		"reboot.target":         "kernel request on Ctrl-Alt-Del",
		"emergency.target":      "on failure of agent.service",
		"emergency.service":     "pulled in by emergency.target",
		"poweroff.target":       "failure action of agent.service",
		"shell@reachable-units-placeholder.service": "accepted by shell.socket",
		"helper.service":      "pulled in by shell@.service",
		"dbus-broker.service": "triggered by dbus.socket",
		"example.service":     "D-Bus activation of org.example",
		"sound.target":        "udev rule in 99-test.rules",
	}, reached)
	assert.Equal(t, []string{
		"boot.target", "ctrl-alt-del.target", "kbrequest.target", "emergency.target", "poweroff.target",
		"dbus-broker.service", "shell@reachable-units-placeholder.service", "sound.target", "dbus-org.example.service",
	}, systemd.started)
}

func TestAnalysisUnusedReviewedCommand(t *testing.T) {
	systemd := &fakeSystemd{units: map[string]*unitInfo{"boot.target": loaded(unitInfo{})}}
	cfg := config{
		Target:           "boot.target",
		ReviewedCommands: map[string][]string{"systemctl start gone.service": {"gone.service"}},
	}
	_, err := newAnalysis(cfg, systemd.transaction).run(t.Context())
	assert.ErrorContains(t, err, "systemctl start gone.service")
}

func TestAnalysisDeviceDeps(t *testing.T) {
	etc := writeFiles(t, map[string]string{"systemd/system/dev-vda.device.wants/foo.service": ""})
	systemd := &fakeSystemd{units: map[string]*unitInfo{"boot.target": loaded(unitInfo{})}}
	_, err := newAnalysis(config{Target: "boot.target", Etc: etc}, systemd.transaction).run(t.Context())
	assert.ErrorContains(t, err, "dev-vda.device.wants")
}

func TestAnalysisUnitActiveInEnvironment(t *testing.T) {
	systemd := &fakeSystemd{
		units: map[string]*unitInfo{
			"boot.target":     loaded(unitInfo{}),
			"nix-store.mount": loaded(unitInfo{activeState: "active"}),
		},
	}
	_, err := newAnalysis(config{Target: "boot.target"}, systemd.transaction).run(t.Context())
	require.NoError(t, err)

	systemd.units["dev-hugepages.mount"] = loaded(unitInfo{activeState: "active", fragmentPath: "/etc/systemd/system/dev-hugepages.mount"})
	_, err = newAnalysis(config{Target: "boot.target"}, systemd.transaction).run(t.Context())
	assert.ErrorContains(t, err, "dev-hugepages.mount")
}
