// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	generators := []string{"systemd-debug-generator", "systemd-fstab-generator"}
	reached := map[string]string{
		"boot.target": reasonBoot,
		"-.slice":     reasonPerpetual,
		"foo@reachable-units-placeholder.service": "accepted by foo.socket",
		"foo.socket": reasonBoot,
	}

	var out strings.Builder
	require.NoError(t, write(&out, config{Name: "system", Target: "boot.target"}, generators, reached))
	assert.Equal(t, `# system (boot.target)
#
# generators:
#   systemd-debug-generator
#   systemd-fstab-generator
#
-.slice (perpetual)
boot.target
foo.socket
foo@.service (accepted by foo.socket)
`, out.String())
}
