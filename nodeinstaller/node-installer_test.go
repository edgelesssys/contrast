// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

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
		"BareMetalQEMUTDX": {
			platform: platforms.K3sQEMUTDX,
			expected: expectedConfBareMetalQEMUTDX,
		},
		"BareMetalQEMUSNP": {
			platform: platforms.K3sQEMUSNP,
			expected: expectedConfBareMetalQEMUSNP,
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

			tmpDir, err := os.MkdirTemp("", "patch-containerd-config-test")
			require.NoError(err)
			t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

			configPath := filepath.Join(tmpDir, "config.toml")

			runtimeHandler := "my-runtime"

			err = patchContainerdConfig(runtimeHandler,
				filepath.Join("/opt/edgeless", runtimeHandler), configPath, tc.platform)
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
