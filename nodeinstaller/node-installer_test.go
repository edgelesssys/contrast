// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"os"
	"path/filepath"
	"testing"

	_ "embed"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/expected-aks-clh-snp.toml
	expectedConfAKSCLHSNP []byte
	//go:embed testdata/expected-bare-metal-qemu-tdx.toml
	expectedConfBareMetalQEMUTDX []byte
	//go:embed testdata/expected-bare-metal-qemu-snp.toml
	expectedConfBareMetalQEMUSNP []byte
	//go:embed testdata/expected-bare-metal-qemu-snp-gpu.toml
	expectedConfBareMetalQEMUSNPGPU []byte
)

func TestPatchContainerdConfig(t *testing.T) {
	testCases := map[string]struct {
		platform platforms.Platform
		expected []byte
		wantErr  bool
	}{
		"AKSCLHSNP": {
			platform: platforms.AKSCloudHypervisorSNP,
			expected: expectedConfAKSCLHSNP,
		},
		"K3sQEMUTDX": {
			platform: platforms.K3sQEMUTDX,
			expected: expectedConfBareMetalQEMUTDX,
		},
		"K3sQEMUSNP": {
			platform: platforms.K3sQEMUSNP,
			expected: expectedConfBareMetalQEMUSNP,
		},
		"K3sQEMUSNPGPU": {
			platform: platforms.K3sQEMUSNPGPU,
			expected: expectedConfBareMetalQEMUSNPGPU,
		},
		"Unknown": {
			platform: platforms.Unknown,
			wantErr:  true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			tmpDir := t.TempDir()
			t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

			configPath := filepath.Join(tmpDir, "config.toml")

			runtimeHandler := "my-runtime"

			err := patchContainerdConfig(runtimeHandler,
				filepath.Join("/opt/edgeless", runtimeHandler), configPath, tc.platform, true)
			if tc.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			configData, err := os.ReadFile(configPath)
			require.NoError(err)
			assert.Equal(string(tc.expected), string(configData))
		})
	}
}
