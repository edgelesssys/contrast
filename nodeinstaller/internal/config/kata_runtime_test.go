// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package config_test

import (
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/constants"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKataConfig(t *testing.T) {
	// This is a regression test that ensures the `agent.kata` section is not optimized away. Empty
	// section and no section are handled differently by Kata, so we make sure that this section is
	// always present.
	for _, platform := range platforms.All() {
		t.Run(platform.String(), func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			cfg, err := constants.KataRuntimeConfig("/", platform, "", sevsnp.SevProduct_SEV_PRODUCT_MILAN, false)
			require.NoError(err)
			configBytes, err := cfg.Marshal()
			require.NoError(err)
			assert.Contains(string(configBytes), "[Agent.kata]")
			assert.Contains(string(configBytes), "[Runtime]")

			switch platform {
			case platforms.K3sQEMUSNP, platforms.K3sQEMUSNPGPU, platforms.K3sQEMUTDX,
				platforms.MetalQEMUSNP, platforms.MetalQEMUTDX, platforms.RKE2QEMUTDX,
				platforms.MetalQEMUSNPGPU:
				assert.Contains(string(configBytes), "[Hypervisor.qemu]")
			case platforms.AKSCloudHypervisorSNP:
				assert.Contains(string(configBytes), "[Hypervisor.clh]")
			default:
				assert.Fail("missing hypervisor test expectations")
			}
		})
	}
}
