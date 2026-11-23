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

func TestUnitActivations(t *testing.T) {
	syncScript := filepath.Join(t.TempDir(), "sync")
	require.NoError(t, os.WriteFile(syncScript, []byte("#!/bin/sh\nsync\n"), 0o755))
	testCases := map[string]struct {
		name    string
		info    *unitInfo
		want    []activation
		wantErr bool
	}{
		"dependencies": {
			name: "foo.service",
			info: &unitInfo{deps: map[string][]string{
				"Upholds":   {"upheld.service"},
				"OnFailure": {"emergency.target", "other.target"},
			}},
			want: []activation{
				{"emergency.target", "on failure of foo.service"},
				{"other.target", "on failure of foo.service"},
				{"upheld.service", "upheld by foo.service"},
			},
		},
		"socket triggering template instance": {
			name: "foo.socket",
			info: &unitInfo{deps: map[string][]string{"Triggers": {"foo@reachable-units-placeholder.service"}}},
			want: []activation{{"foo@reachable-units-placeholder.service", "triggered by foo.socket"}},
		},
		"accepting socket": {
			name: "foo.socket",
			info: &unitInfo{accept: true},
			want: []activation{{"foo@reachable-units-placeholder.service", "accepted by foo.socket"}},
		},
		"emergency actions": {
			name: "foo@reachable-units-placeholder.service",
			info: &unitInfo{actions: map[string]string{
				"Failure Action":     "poweroff",
				"Start Limit Action": "reboot-force",
				"Success Action":     "none",
			}},
			// reboot-force.target doesn't exist, so the analysis skips it.
			want: []activation{
				{"poweroff.target", "failure action of foo@.service"},
				{"reboot-force.target", "start limit action of foo@.service"},
			},
		},
		"reviewed command": {
			name: "foo.service",
			info: &unitInfo{commands: []string{syncScript, "systemctl --no-block isolate bar.target"}},
			want: []activation{{"bar.target", "started by command of foo.service"}},
		},
		"command not reviewed": {
			name:    "foo.service",
			info:    &unitInfo{commands: []string{"systemctl start baz.service"}},
			wantErr: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			reviewed := newReviewedCommands(map[string][]string{"systemctl --no-block isolate bar.target": {"bar.target"}})
			got, err := unitActivations(tc.name, tc.info, reviewed)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
