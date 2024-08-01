// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "embed"

	"github.com/edgelesssys/contrast/internal/manifest"
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

	//go:embed testdata/input-bare-metal-qemu-tdx.toml.tmpl
	inputConfTmplBareMetalQEMUTDX []byte
	//go:embed testdata/expected-bare-metal-qemu-tdx.toml.tmpl
	expectedConfTmplBareMetalQEMUTDX []byte
	//go:embed testdata/input-bare-metal-qemu-snp.toml.tmpl
	inputConfTmplBareMetalQEMUSNP []byte
	//go:embed testdata/expected-bare-metal-qemu-snp.toml.tmpl
	expectedConfTmplBareMetalQEMUSNP []byte
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
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			tmpDir, err := os.MkdirTemp("", "patch-containerd-config-test")
			require.NoError(err)
			t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

			configPath := filepath.Join(tmpDir, "config.toml")

			runtimeHandler, err := manifest.RuntimeHandler(tc.platform)
			require.NoError(err)

			err = patchContainerdConfig(filepath.Join("/opt/edgeless", runtimeHandler),
				configPath, tc.platform)
			if tc.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			configData, err := os.ReadFile(configPath)
			require.NoError(err)
			expected := strings.ReplaceAll(string(tc.expected), "RUNTIMEHANDLER", runtimeHandler)
			fmt.Println("expected: ", expected)
			assert.Equal(expected, string(configData))
		})
	}
}

func TestPatchContainerdConfigTemplate(t *testing.T) {
	testCases := map[string]struct {
		platform platforms.Platform
		input    []byte
		expected []byte
	}{
		"BareMetalQEMUTDX": {
			platform: platforms.K3sQEMUTDX,
			input:    inputConfTmplBareMetalQEMUTDX,
			expected: expectedConfTmplBareMetalQEMUTDX,
		},
		"BareMetalQEMUSNP": {
			platform: platforms.K3sQEMUSNP,
			input:    inputConfTmplBareMetalQEMUSNP,
			expected: expectedConfTmplBareMetalQEMUSNP,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			tmpDir, err := os.MkdirTemp("", "patch-containerd-config-test")
			require.NoError(err)
			t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

			// Unlike patchContainerdConfig, patchContainerdConfigTemplate
			// requires the file to exist already. Create one.
			configTemplatePath := filepath.Join(tmpDir, "config.toml.tmpl")
			err = os.WriteFile(configTemplatePath, tc.input, os.ModePerm)
			require.NoError(err)

			// Testing patching a config template.

			runtimeHandler, err := manifest.RuntimeHandler(tc.platform)
			require.NoError(err)

			err = patchContainerdConfigTemplate(filepath.Join("/opt/edgeless", runtimeHandler),
				configTemplatePath, tc.platform)
			require.NoError(err)

			configData, err := os.ReadFile(configTemplatePath)
			require.NoError(err)
			expected := strings.ReplaceAll(string(tc.expected), "RUNTIMEHANDLER", runtimeHandler)
			assert.Equal(expected, string(configData))

			// Test that patching the same template twice doesn't change it.

			err = patchContainerdConfigTemplate("/opt/edgeless/my-runtime",
				configTemplatePath, tc.platform)
			require.NoError(err)

			configData, err = os.ReadFile(configTemplatePath)
			require.NoError(err)
			assert.Equal(expected, string(configData))
		})
	}
}
