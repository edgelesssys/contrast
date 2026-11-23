// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Excerpt of the output of `systemd --test`, see parseDump.
const testDump = "-> By units:\n" +
	"\t→ Unit kata-agent.service:\n" +
	"\t\tDescription: Kata Containers Agent\n" +
	"\t\tUnit Load State: loaded\n" +
	"\t\tUnit Active State: inactive\n" +
	"\t\tPerpetual: no\n" +
	"\t\tFragment Path: /etc/systemd/system/kata-agent.service\n" +
	"\t\tAlias: agent.service\n" +
	"\t\tOnFailure: emergency.target (origin-file)\n" +
	"\t\tOnFailure: other.target (origin-file)\n" +
	"\t\tFailure Action: poweroff\n" +
	"\t\t→ ExecStart:\n" +
	"\t\t\tCommand Line: /bin/kata-agent --config /run/agent.toml\n" +
	"\t\t→ ExecStop:\n" +
	"\t\t\tCommand Line: /bin/sync\n" +
	"\t\t\tCommand Line: /bin/systemctl --force poweroff\n" +
	"\t\t→ Job 1:\n" +
	"\t\t\tAction: kata-agent.service -> start\n" +
	"\t→ Unit system.slice:\n" +
	"\t\tUnit Load State: loaded\n" +
	"\t\tUnit Active State: active\n" +
	"\t\tPerpetual: yes\n" +
	"\t→ Unit foo.socket:\n" +
	"\t\tUnit Load State: loaded\n" +
	"\t\tTriggers: foo@reachable-units-placeholder.service (origin-implicit)\n" +
	"\t\tAccepted: 0\n" +
	"\t→ Unit missing.service:\n" +
	"\t\tUnit Load State: not-found\n" +
	"-> By jobs:\n" +
	"\t→ Job 1:\n" +
	"\t\tAction: kata-agent.service -> start\n" +
	"\t→ Job 2:\n" +
	"\t\tAction: foo.socket -> start\n" +
	"\t→ Job 3:\n" +
	"\t\tAction: bar.service -> stop\n"

func TestParseDump(t *testing.T) {
	d := parseDump(testDump)

	assert.Equal(t, []string{"kata-agent.service", "foo.socket"}, d.jobs)
	assert.Equal(t, map[string]string{"agent.service": "kata-agent.service"}, d.aliases)
	require.Len(t, d.units, 4)
	assert.Equal(t, &unitInfo{
		loadState:    "loaded",
		activeState:  "inactive",
		fragmentPath: "/etc/systemd/system/kata-agent.service",
		deps:         map[string][]string{"OnFailure": {"emergency.target", "other.target"}},
		actions:      map[string]string{"Failure Action": "poweroff"},
		commands:     []string{"/bin/kata-agent --config /run/agent.toml", "/bin/sync", "/bin/systemctl --force poweroff"},
	}, d.units["kata-agent.service"])
	assert.True(t, d.units["system.slice"].perpetual)
	assert.Equal(t, "active", d.units["system.slice"].activeState)
	assert.True(t, d.units["foo.socket"].accept)
	assert.Equal(t, []string{"foo@reachable-units-placeholder.service"}, d.units["foo.socket"].deps["Triggers"])
	assert.Equal(t, "not-found", d.units["missing.service"].loadState)
}
