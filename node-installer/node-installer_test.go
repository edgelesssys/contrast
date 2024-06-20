// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"os"
	"path/filepath"
	"testing"

	_ "embed"

	"github.com/edgelesssys/contrast/node-installer/flavours"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/expected-aks-clh-snp.toml
	expectedConfAKSCLHSNP []byte

	//go:embed testdata/expected-bare-metal-qemu-tdx.toml
	expectedConfBareMetalQEMUTDX []byte
)

func TestPatchContainerdConfig(t *testing.T) {
	testCases := map[string]struct {
		flavour  flavours.Flavour
		expected []byte
		wantErr  bool
	}{
		"AKSCLHSNP": {
			flavour:  flavours.AKSCLHSNP,
			expected: expectedConfAKSCLHSNP,
		},
		"BareMetalQEMUTDX": {
			flavour:  flavours.BareMetalQEMUTDX,
			expected: expectedConfBareMetalQEMUTDX,
		},
		"Unknown": {
			flavour: flavours.Unknown,
			wantErr: true,
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

			err = patchContainerdConfig("my-runtime", "/opt/edgeless/my-runtime",
				configPath, tc.flavour)
			if tc.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			configData, err := os.ReadFile(configPath)
			require.NoError(err)
			assert.Equal(tc.expected, configData)
		})
	}
}
