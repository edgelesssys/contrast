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
		wantErr  bool
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
