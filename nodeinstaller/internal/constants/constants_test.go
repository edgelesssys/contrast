// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package constants_test

import (
	_ "embed"
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/expected-configuration-clh-snp.toml
	expectedConfAKSCLHSNP []byte
	//go:embed testdata/expected-configuration-qemu-snp.toml
	expectedConfBareMetalQEMUSNP []byte
	//go:embed testdata/expected-configuration-qemu-tdx.toml
	expectedConfBareMetalQEMUTDX []byte
	//go:embed testdata/expected-configuration-qemu-snp-gpu.toml
	expectedConfBareMetalQEMUSNPGPU []byte
)

func TestKataRuntimeConfig(t *testing.T) {
	testCases := map[platforms.Platform]struct {
		changeSnpFields bool
		want            string
	}{
		platforms.AKSCloudHypervisorSNP: {
			changeSnpFields: false,
			want:            string(expectedConfAKSCLHSNP),
		},
		platforms.K3sQEMUSNP: {
			changeSnpFields: true,
			want:            string(expectedConfBareMetalQEMUSNP),
		},
		platforms.K3sQEMUSNPGPU: {
			changeSnpFields: true,
			want:            string(expectedConfBareMetalQEMUSNPGPU),
		},
		platforms.K3sQEMUTDX: {
			changeSnpFields: false,
			want:            string(expectedConfBareMetalQEMUTDX),
		},
	}
	for platform, tc := range testCases {
		t.Run(platform.String(), func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			snpIDBlock := constants.SnpIDBlock{
				IDAuth:  "PLACEHOLDER_ID_AUTH",
				IDBlock: "PLACEHOLDER_ID_BLOCK",
			}
			cfg, err := constants.KataRuntimeConfig("/", platform, "", snpIDBlock, false)
			require.NoError(err)

			configBytes, err := cfg.Marshal()
			require.NoError(err)

			assert.Equal(tc.want, string(configBytes))

			// This is a regression test that ensures the `agent.kata` section is not optimized away. Empty
			// section and no section are handled differently by Kata, so we make sure that this section is
			// always present.
			// It's covered by the comparison with testdata, but we want to keep this explicit.
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
