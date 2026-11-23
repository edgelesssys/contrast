// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUdevActivations(t *testing.T) {
	vconsoleRule := `ACTION=="add", SUBSYSTEM=="vtconsole", RUN+="systemctl --no-block restart systemd-vconsole-setup.service"`
	libexecDir := t.TempDir()
	helper := filepath.Join(libexecDir, "start-helper")
	require.NoError(t, os.WriteFile(helper, []byte("#!/bin/sh\nsystemctl start extra.service\n"), 0o755))
	reviewedHelperRule := `ACTION=="change", RUN{program}+="` + helper + ` --now"`
	missingHelperRule := `ACTION=="add", RUN+="/run/start-helper"`
	wantsFile := filepath.Join(libexecDir, "wants")
	require.NoError(t, os.WriteFile(wantsFile, []byte("SYSTEMD_WANTS=extra.service\n"), 0o644))
	otherFile := filepath.Join(libexecDir, "other")
	require.NoError(t, os.WriteFile(otherFile, []byte("FOO=1\n"), 0o644))
	exportWants := filepath.Join(libexecDir, "export-wants")
	require.NoError(t, os.WriteFile(exportWants, []byte("#!/bin/sh\necho SYSTEMD_WANTS=extra.service\n"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(libexecDir, "ata_id"), []byte("\x7fELF ID_ATA=1"), 0o755))

	testCases := map[string]struct {
		rules   string
		want    []activation
		wantErr bool
	}{
		"SYSTEMD_WANTS": {
			rules: `SUBSYSTEM=="rfkill", ENV{SYSTEMD_WANTS}+="systemd-rfkill.socket"
SUBSYSTEM=="backlight", ENV{SYSTEMD_WANTS}+="systemd-backlight@backlight:$name.service sound.target"`,
			want: []activation{
				{"systemd-rfkill.socket", "udev rule in 99-test.rules"},
				{"systemd-backlight@reachable-units-placeholder.service", "udev rule in 99-test.rules"},
				{"sound.target", "udev rule in 99-test.rules"},
			},
		},
		"SYSTEMD_WANTS with whitespace and explicit instance": {
			rules: `ENV{SYSTEMD_WANTS} += "extra.service", ENV{SYSTEMD_WANTS} := "worker@disk.service"`,
			want: []activation{
				{"extra.service", "udev rule in 99-test.rules"},
				{"worker@disk.service", "udev rule in 99-test.rules"},
			},
		},
		"reviewed systemctl": {
			rules: "  " + vconsoleRule,
			want:  []activation{{"systemd-vconsole-setup.service", "udev rule in 99-test.rules"}},
		},
		"continued SYSTEMD_WANTS": {
			rules: "ENV{SYSTEMD_WANTS}+=\"alpha.service \\\n  beta.service\"",
			want: []activation{
				{"alpha.service", "udev rule in 99-test.rules"},
				{"beta.service", "udev rule in 99-test.rules"},
			},
		},
		"escaped SYSTEMD_WANTS": {
			rules: `TAG+="systemd", ENV{SYSTEMD_WANTS}+=e"extra.service\tother\x2eservice"`,
			want: []activation{
				{"extra.service", "udev rule in 99-test.rules"},
				{"other.service", "udev rule in 99-test.rules"},
			},
		},
		"escaped quotes in SYSTEMD_WANTS": {
			rules: `ENV{SYSTEMD_WANTS}+="alpha.service \"beta.service\" 'gamma'.service" ENV{SYSTEMD_WANTS}+="dev-disk-by\x2dlabel-foo.device"`,
			want: []activation{
				{"alpha.service", "udev rule in 99-test.rules"},
				{"beta.service", "udev rule in 99-test.rules"},
				{"gamma.service", "udev rule in 99-test.rules"},
				{`dev-disk-by\x2dlabel-foo.device`, "udev rule in 99-test.rules"},
			},
		},
		"unbalanced quotes in SYSTEMD_WANTS": {
			rules:   `ENV{SYSTEMD_WANTS}+="'extra.service"`,
			wantErr: true,
		},
		"shorthand SYSTEMD_WANTS": {
			rules:   `ENV{SYSTEMD_WANTS}+="extra"`,
			wantErr: true,
		},
		"helper in rule with case insensitive match": {
			rules:   `SUBSYSTEM==i"USB", PROGRAM=="` + helper + `"`,
			wantErr: true,
		},
		"undecodable SYSTEMD_WANTS": {
			rules:   `ENV{SYSTEMD_WANTS}+=e"extra\qservice"`,
			wantErr: true,
		},
		"helper in escaped RUN": {
			rules:   `RUN+=e"` + helper + `"`,
			wantErr: true,
		},
		"continued comment": {
			rules: "# a comment \\\nENV{SYSTEMD_WANTS}+=\"extra.service\"",
			want:  []activation{{"extra.service", "udev rule in 99-test.rules"}},
		},
		"helper in PROGRAM": {
			rules:   `PROGRAM=="` + helper + `", ENV{FOO}="1"`,
			wantErr: true,
		},
		"helper in IMPORT": {
			rules:   `IMPORT{program}="` + helper + ` $devnode"`,
			wantErr: true,
		},
		"missing helper": {
			rules:   `ACTION=="add", RUN+="/run/other-helper"`,
			wantErr: true,
		},
		"helper passed to interpreter": {
			rules:   `RUN+="/bin/sh ` + helper + `"`,
			wantErr: true,
		},
		"IMPORT cmdline SYSTEMD_WANTS": {
			rules:   `TAG+="systemd", IMPORT{cmdline}="SYSTEMD_WANTS"`,
			wantErr: true,
		},
		"IMPORT all properties of parent": {
			rules:   `TAG+="systemd", IMPORT{parent}="*"`,
			wantErr: true,
		},
		"IMPORT alternatives with SYSTEMD_WANTS": {
			rules:   `TAG+="systemd", IMPORT{parent}="ID_*|SYSTEMD_WANTS"`,
			wantErr: true,
		},
		"IMPORT parent with substitution": {
			rules:   `TAG+="systemd", IMPORT{parent}="$env{IMPORT_KEY}"`,
			wantErr: true,
		},
		"IMPORT other properties": {
			rules: `IMPORT{cmdline}="nomodeset", IMPORT{parent}="ID_*", IMPORT{db}="ID_PATH"`,
		},
		"hwdb without SYSTEMD_WANTS": {
			rules: `IMPORT{builtin}="hwdb --subsystem=usb"`,
		},
		"reviewed missing helper": {
			rules: missingHelperRule,
			want:  []activation{{"extra.service", "udev rule in 99-test.rules"}},
		},
		"comments": {
			rules: `# ENV{SYSTEMD_WANTS}+="extra.service"
  # RUN+="systemctl start extra.service"`,
		},
		"helper script with systemctl": {
			rules:   `ACTION=="add", RUN+="` + helper + `"`,
			wantErr: true,
		},
		"reviewed helper script": {
			rules: reviewedHelperRule,
			want:  []activation{{"extra.service", "udev rule in 99-test.rules"}},
		},
		"other programs": {
			rules: `PROGRAM=="ata_id --check", IMPORT{program}="ata_id --export $devnode", RUN{builtin}+="kmod load $env{MODALIAS}"`,
		},
		"SYSTEMD_WANTS with variable": {
			rules:   `ENV{SYSTEMD_WANTS}+="$env{UNIT}"`,
			wantErr: true,
		},
		"bare helper in PROGRAM": {
			rules:   `PROGRAM=="start-helper", ENV{FOO}="1"`,
			wantErr: true,
		},
		"IMPORT file with SYSTEMD_WANTS": {
			rules:   `TAG+="systemd", IMPORT{file}="` + wantsFile + `"`,
			wantErr: true,
		},
		"IMPORT missing file": {
			rules:   `TAG+="systemd", IMPORT{file}="/run/wants"`,
			wantErr: true,
		},
		"IMPORT program with SYSTEMD_WANTS": {
			rules:   `TAG+="systemd", IMPORT{program}="` + exportWants + ` $devnode"`,
			wantErr: true,
		},
		"IMPORT inline command with SYSTEMD_WANTS": {
			rules:   `TAG+="systemd", IMPORT{program}="/bin/sh -c 'echo SYSTEMD_WANTS=extra.service'"`,
			wantErr: true,
		},
		"IMPORT script with SYSTEMD_WANTS passed to interpreter": {
			rules:   `TAG+="systemd", IMPORT{program}="/bin/sh ` + exportWants + `"`,
			wantErr: true,
		},
		"IMPORT missing program": {
			rules:   `TAG+="systemd", IMPORT{program}="/run/export-wants"`,
			wantErr: true,
		},
		"IMPORT file without SYSTEMD_WANTS": {
			rules: `IMPORT{file}="` + otherFile + `"`,
		},
		"systemctl not reviewed": {
			rules:   `RUN+="/bin/sh -c 'systemctl start foo.service'"`,
			wantErr: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "99-test.rules"), []byte(tc.rules+"\n"), 0o644))
			reviewed := newReviewedCommands(map[string][]string{
				vconsoleRule:       {"systemd-vconsole-setup.service"},
				reviewedHelperRule: {"extra.service"},
				missingHelperRule:  {"extra.service"},
			})
			got, err := udevActivations(dir, libexecDir, reviewed)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestUdevActivationsHwdb(t *testing.T) {
	etcUdev := t.TempDir()
	rulesDir := filepath.Join(etcUdev, "rules.d")
	require.NoError(t, os.Mkdir(rulesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "99-test.rules"), []byte(`IMPORT{builtin}="hwdb --subsystem=usb"`+"\n"), 0o644))

	_, err := udevActivations(rulesDir, t.TempDir(), newReviewedCommands(nil))
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(etcUdev, "hwdb.bin"), []byte("KSLPHHRH\x00SYSTEMD_WANTS\x00extra.service"), 0o444))
	_, err = udevActivations(rulesDir, t.TempDir(), newReviewedCommands(nil))
	assert.Error(t, err)
}

func TestUdevActivationsUnreadableHwdb(t *testing.T) {
	etcUdev := t.TempDir()
	rulesDir := filepath.Join(etcUdev, "rules.d")
	require.NoError(t, os.Mkdir(rulesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "99-test.rules"), []byte(`IMPORT{builtin}="hwdb --subsystem=usb"`+"\n"), 0o644))
	// Reading a directory fails regardless of permissions.
	require.NoError(t, os.Mkdir(filepath.Join(etcUdev, "hwdb.bin"), 0o755))

	_, err := udevActivations(rulesDir, t.TempDir(), newReviewedCommands(nil))
	assert.Error(t, err)
}

func TestUdevActivationsHwdbSource(t *testing.T) {
	etcUdev := t.TempDir()
	rulesDir := filepath.Join(etcUdev, "rules.d")
	require.NoError(t, os.MkdirAll(filepath.Join(etcUdev, "hwdb.d"), 0o755))
	require.NoError(t, os.Mkdir(rulesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "99-test.rules"), []byte(`IMPORT{builtin}="hwdb --subsystem=usb"`+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(etcUdev, "hwdb.d/99-test.hwdb"), []byte("usb:v1234*\n SYSTEMD_WANTS=extra.service\n"), 0o644))

	_, err := udevActivations(rulesDir, t.TempDir(), newReviewedCommands(nil))
	assert.Error(t, err)
}
