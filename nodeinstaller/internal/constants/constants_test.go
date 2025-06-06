// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package constants_test

import (
	_ "embed"
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/constants"
	"github.com/google/go-sev-guest/proto/sevsnp"
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
)

func TestKataRuntimeConfig(t *testing.T) {
	testCases := map[string]struct {
		platform        platforms.Platform
		changeSnpFields bool
		testdata        []byte
	}{
		"AKSCLHSNP": {
			platform:        platforms.AKSCloudHypervisorSNP,
			changeSnpFields: false,
			testdata:        expectedConfAKSCLHSNP,
		},
		"K3sQEMUSNP": {
			platform:        platforms.K3sQEMUSNP,
			changeSnpFields: true,
			testdata:        expectedConfBareMetalQEMUSNP,
		},
		"K3sQEMUTDX": {
			platform:        platforms.K3sQEMUTDX,
			changeSnpFields: false,
			testdata:        expectedConfBareMetalQEMUTDX,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			cfg, err := constants.KataRuntimeConfig("/", tc.platform, "", sevsnp.SevProduct_SEV_PRODUCT_MILAN, false)

			// Avoids having to manually obtain these values whenever they change.
			if tc.changeSnpFields {
				cfg.Hypervisor["qemu"]["snp_id_auth"] = "PLACEHOLDER_ID_AUTH"
				cfg.Hypervisor["qemu"]["snp_id_block"] = "PLACEHOLDER_ID_BLOCK"
			}

			require.NoError(err)
			configBytes, err := cfg.Marshal()
			require.NoError(err)

			assert.Equal(configBytes, tc.testdata)
		})
	}
}
