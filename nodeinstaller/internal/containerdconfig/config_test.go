// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package containerdconfig

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/containerd-config.toml
	exemplaryContainerConfig []byte
	//go:embed testdata/expected-bare-metal-qemu-tdx.toml
	expectedConfMetalQEMUTDX []byte
	//go:embed testdata/expected-bare-metal-qemu-snp.toml
	expectedConfMetalQEMUSNP []byte
	//go:embed testdata/expected-bare-metal-qemu-snp-gpu.toml
	expectedConfMetalQEMUSNPGPU []byte
)

func TestPatchContainerdConfig(t *testing.T) {
	testCases := map[string]struct {
		platform platforms.Platform
		expected []byte
	}{
		"MetalQEMUTDX": {
			platform: platforms.MetalQEMUTDX,
			expected: expectedConfMetalQEMUTDX,
		},
		"MetalQEMUSNP": {
			platform: platforms.MetalQEMUSNP,
			expected: expectedConfMetalQEMUSNP,
		},
		"MetalQEMUSNPGPU": {
			platform: platforms.MetalQEMUSNPGPU,
			expected: expectedConfMetalQEMUSNPGPU,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			configPath := filepath.Join(t.TempDir(), "config.toml")
			require.NoError(os.WriteFile(configPath, exemplaryContainerConfig, 0o644))

			runtimeHandler := "my-runtime"
			runtimeBaseDir := filepath.Join("/opt/edgeless", runtimeHandler)

			conf, err := FromPath(configPath)
			require.NoError(err)
			conf.EnableDebug()
			runtimeFragment, err := ContrastRuntime(runtimeBaseDir, tc.platform)
			require.NoError(err)
			conf.AddRuntime(runtimeHandler, runtimeFragment)
			require.NoError(conf.Write())

			configData, err := os.ReadFile(configPath)
			require.NoError(err)
			assert.Equal(string(tc.expected), string(configData))
		})
	}
}
