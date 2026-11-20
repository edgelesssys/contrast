// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBusActivations(t *testing.T) {
	testCases := map[string]struct {
		files   map[string]string
		want    []activation
		wantErr bool
	}{
		"services": {
			files: map[string]string{
				"a/org.freedesktop.systemd1.service": "[D-BUS Service]\nName=org.freedesktop.systemd1\nExec=/bin/false\n",
				"a/org.example.A.service":            "[D-BUS Service]\nName=org.example.A\nExec=/bin/false\nSystemdService=dbus-org.example.A.service\n",
				"b/org.example.B.service":            "# comment\n[D-BUS Service]\nName=org.example.B\nSystemdService=b.service\nUser=root\n",
				"b/org.example.C.service":            "[D-BUS Service]\nName = org.example.C\nSystemdService =  c.service \n",
				"b/not-a-service.conf":               "",
				"b/shadowed-a.service":               "[D-BUS Service]\nName=org.example.A\nSystemdService=shadowed.service\n",
				"b/shadowed-a-without-unit.service":  "[D-BUS Service]\nName=org.example.A\nExec=/bin/false\n",
			},
			want: []activation{
				{"dbus-org.example.A.service", "D-Bus activation of org.example.A"},
				{"b.service", "D-Bus activation of org.example.B"},
				{"c.service", "D-Bus activation of org.example.C"},
			},
		},
		"service without SystemdService": {
			files:   map[string]string{"a/org.example.A.service": "[D-BUS Service]\nName=org.example.A\nExec=/bin/false\n"},
			wantErr: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			dir := writeFiles(t, tc.files)
			// Missing directories are ignored, like by the bus daemon.
			dirs := []string{filepath.Join(dir, "a"), filepath.Join(dir, "b"), filepath.Join(dir, "missing")}
			got, err := dbusActivations(dirs)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
