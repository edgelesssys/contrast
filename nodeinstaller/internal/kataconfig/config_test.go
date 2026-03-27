// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kataconfig_test

import (
	_ "embed"
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/kataconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKataRuntimeConfig(t *testing.T) {
	testCases := map[platforms.Platform]struct {
		changeSnpFields bool
		want            string
	}{
		platforms.MetalQEMUSNP: {
			changeSnpFields: true,
			want:            string(expectedConfMetalQEMUSNP),
		},
		platforms.MetalQEMUSNPGPU: {
			changeSnpFields: true,
			want:            string(expectedConfMetalQEMUSNPGPU),
		},
		platforms.MetalQEMUTDX: {
			changeSnpFields: false,
			want:            string(expectedConfMetalQEMUTDX),
		},
		platforms.MetalQEMUTDXGPU: {
			changeSnpFields: false,
			want:            string(expectedConfMetalQEMUTDXGPU),
		},
	}
	for platform, tc := range testCases {
		t.Run(platform.String(), func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			cfg, err := kataconfig.KataRuntimeConfig("/", platform, "", "", false)
			require.NoError(err)

			configBytes, err := cfg.Marshal()
			require.NoError(err)

			assert.Equal(tc.want, string(configBytes))

			// This is a regression test that ensures the `agent.kata` section is not optimized away. Empty
			// section and no section are handled differently by Kata, so we make sure that this section is
			// always present.
			// It's covered by the comparison with testdata, but we want to keep this explicit.
			assert.Contains(string(configBytes), "[agent.kata]")
			assert.Contains(string(configBytes), "[runtime]")
			assert.Contains(string(configBytes), "[hypervisor.qemu]")
		})
	}
}
