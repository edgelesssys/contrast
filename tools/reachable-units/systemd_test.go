// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGenerators(t *testing.T) {
	// Excerpt of the debug log of `systemd --test`.
	log := `File '/etc/systemd/system-generators/systemd-gpt-auto-generator' is a mask (symlink to /dev/null).
Successfully forked off '(generators)' as PID 581298.
About to execute /lib/systemd/system-generators/systemd-fstab-generator
About to execute /lib/systemd/system-generators/systemd-debug-generator
'/lib/systemd/system-generators/systemd-debug-generator' succeeded.
'/lib/systemd/system-generators/systemd-fstab-generator' succeeded.
'(generators)' succeeded.
`
	generators, err := parseGenerators(log)
	require.NoError(t, err)
	assert.Equal(t, []string{"systemd-debug-generator", "systemd-fstab-generator"}, generators)

	failed := `About to execute /lib/systemd/system-generators/systemd-fstab-generator
'/lib/systemd/system-generators/systemd-fstab-generator' failed with exit status 1.
`
	_, err = parseGenerators(failed)
	assert.ErrorContains(t, err, "systemd-fstab-generator")
}
