// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewedCommands(t *testing.T) {
	dir := t.TempDir()
	script := func(name, content string) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
		return path
	}
	restartFoo := script("restart-foo", "#!/bin/sh\nsystemctl restart foo\n")
	echoFoo := script("echo-foo", "#!/bin/sh\necho foo\n")
	binary := script("binary", "\x7fELF systemctl")
	busctlFoo := script("busctl-foo", "#!/bin/sh\nbusctl call org.freedesktop.systemd1 /org/freedesktop/systemd1 org.freedesktop.systemd1.Manager StartUnit ss foo.service replace\n")

	reviewed := newReviewedCommands(map[string][]string{
		"systemctl --no-block isolate foo.target":         {"foo.target"},
		"systemctl --force poweroff":                      {},
		restartFoo:                                        {"foo.service"},
		`RUN+="systemctl --no-block restart bar.service"`: {"bar.service"},
		"systemctl start never-encountered.service":       {"never-encountered.service"},
		"/run/start-helper":                               {"extra.service"},
	})

	testCases := map[string]struct {
		commandLine string
		want        []string
		wantErr     bool
	}{
		"systemctl in store": {
			commandLine: "/nix/store/0123456789abcdfghijklmnpqrsvwxyz-systemd-261.2/bin/systemctl --no-block isolate foo.target",
			want:        []string{"foo.target"},
		},
		"starting nothing":             {commandLine: "/nix/store/0123456789abcdfghijklmnpqrsvwxyz-systemd-261.2/bin/systemctl --force poweroff", want: []string{}},
		"script with systemctl":        {commandLine: restartFoo, want: []string{"foo.service"}},
		"udev rule":                    {commandLine: `RUN+="systemctl --no-block restart bar.service"`, want: []string{"bar.service"}},
		"script without systemctl":     {commandLine: echoFoo + " --some-arg"},
		"binary mentioning systemctl":  {commandLine: binary},
		"script calling D-Bus":         {commandLine: busctlFoo, wantErr: true},
		"missing program":              {commandLine: "/does/not/exist --now", wantErr: true},
		"program in search path":       {commandLine: "modprobe -abq configfs"},
		"relative program":             {commandLine: "./echo-foo", wantErr: true},
		"script passed to interpreter": {commandLine: "/bin/sh " + restartFoo, wantErr: true},
		"device as argument":           {commandLine: echoFoo + " /dev/zero"},
		"systemd-run":                  {commandLine: "systemd-run --property=Wants=foo.service true", wantErr: true},
		"D-Bus call":                   {commandLine: echoFoo + " org.freedesktop.systemd1 StartUnit", wantErr: true},
		"reviewed missing program":     {commandLine: "/run/start-helper", want: []string{"extra.service"}},
		"empty":                        {commandLine: ""},
		"not reviewed":                 {commandLine: "systemctl start foo.service", wantErr: true},
		"indirect":                     {commandLine: `/bin/sh -c "systemctl start foo.service"`, wantErr: true},
		"changed script arguments":     {commandLine: restartFoo + " --now", wantErr: true},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := reviewed.unitsStartedBy(tc.commandLine)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
	assert.Equal(t, []string{"systemctl start never-encountered.service"}, reviewed.unused())

	_, err := reviewed.unitsStartedBy("/nix/store/abcdfghijklmnpqrsvwxyz0123456789-other-1.0/bin/systemctl --no-block isolate foo.target")
	assert.Error(t, err, "same command line of a different package")
}

func TestWithoutStorePaths(t *testing.T) {
	testCases := map[string]string{
		"/nix/store/0123456789abcdfghijklmnpqrsvwxyz-systemd-261.2/bin/systemctl start foo": "systemctl start foo",
		"/nix/store/0123456789abcdfghijklmnpqrsvwxyz-foo/libexec/foo --now":                 "libexec/foo --now",
		"/nix/store/0123456789abcdfghijklmnpqrsvwxyz-restart-foo --now":                     "restart-foo --now",
		`RUN+="/nix/store/0123456789abcdfghijklmnpqrsvwxyz-restart-foo"`:                    `RUN+="restart-foo"`,
		"systemctl --no-block isolate foo.target":                                           "systemctl --no-block isolate foo.target",
	}
	for commandLine, want := range testCases {
		assert.Equal(t, want, withoutStorePaths(commandLine), commandLine)
	}
}

func TestScriptStartsUnitsReferences(t *testing.T) {
	dir := t.TempDir()
	orig := reScriptRef
	reScriptRef = regexp.MustCompile(regexp.QuoteMeta(dir) + `/[^\s"]+`)
	t.Cleanup(func() { reScriptRef = orig })
	script := func(name, content string) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
		return path
	}
	wrapped := script(".foo-wrapped", "#!/bin/python\nsubprocess.run(['systemctl', 'start', 'foo.service'])\n")
	wrapper := script("foo", "#!/bin/sh\nexec \""+wrapped+"\" \"$@\"\n")
	loopA := filepath.Join(dir, "loop-a")
	loopB := script("loop-b", "#!/bin/sh\n"+loopA+"\n")
	script("loop-a", "#!/bin/sh\n"+loopB+"\n")
	missingRef := script("missing-ref", "#!/bin/sh\n"+filepath.Join(dir, "missing")+"\n")

	startsUnits, ok := scriptStartsUnits(wrapper)
	assert.True(t, ok)
	assert.True(t, startsUnits, "wrapped script")
	startsUnits, ok = scriptStartsUnits(loopA)
	assert.True(t, ok)
	assert.False(t, startsUnits, "reference loop")
	startsUnits, ok = scriptStartsUnits(missingRef)
	assert.True(t, ok)
	assert.False(t, startsUnits, "missing reference")
}
