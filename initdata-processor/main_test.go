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

func TestAllowInsecureAttestation(t *testing.T) {
	oldKernelCmdlinePath := kernelCmdlinePath
	t.Cleanup(func() { kernelCmdlinePath = oldKernelCmdlinePath })

	testCases := map[string]struct {
		cmdline string
		want    bool
	}{
		"absent": {
			cmdline: "console=hvc0",
			want:    false,
		},
		"disabled": {
			cmdline: "console=hvc0 contrast.allow_insecure_attestation=0",
			want:    false,
		},
		"enabled": {
			cmdline: "console=hvc0 contrast.allow_insecure_attestation=1 quiet",
			want:    true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			kernelCmdlinePath = filepath.Join(t.TempDir(), "cmdline")
			require.NoError(t, os.WriteFile(kernelCmdlinePath, []byte(tc.cmdline), 0o644))

			got, err := allowInsecureAttestation()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
