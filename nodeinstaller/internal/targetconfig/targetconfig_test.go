// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package targetconfig

import (
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTargetConfig(t *testing.T) {
	testCases := map[string]struct {
		hostMount   string
		runtimeBase string
		platform    platforms.Platform
		wantErr     bool
		wantConfig  *TargetConfig
	}{
		"valid config for metal qemu snp": {
			hostMount:   "/host",
			runtimeBase: "/opt/edgeless/qemu",
			platform:    platforms.MetalQEMUSNP,
			wantErr:     false,
			wantConfig: &TargetConfig{
				containerdConfigPath: "etc/containerd/config.toml",
				restartSystemdUnit:   true,
				systemdUnitNames:     []string{"containerd.service"},
				kataConfigPath:       "/opt/edgeless/qemu/etc/configuration-qemu-snp.toml",
				hostMount:            "/host",
			},
		},
		"valid config for metal qemu tdx": {
			hostMount:   "/host",
			runtimeBase: "/opt/edgeless/qemu",
			platform:    platforms.MetalQEMUTDX,
			wantErr:     false,
			wantConfig: &TargetConfig{
				containerdConfigPath: "etc/containerd/config.toml",
				restartSystemdUnit:   true,
				systemdUnitNames:     []string{"containerd.service"},
				kataConfigPath:       "/opt/edgeless/qemu/etc/configuration-qemu-tdx.toml",
				hostMount:            "/host",
			},
		},
		"invalid platform": {
			hostMount:   "/host",
			runtimeBase: "/opt/edgeless/unknown",
			platform:    platforms.Unknown,
			wantErr:     true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotConfig, err := NewTargetConfig(tc.hostMount, tc.runtimeBase, tc.platform)
			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			gotConfig.fs = nil
			assert.Equal(tc.wantConfig, gotConfig)
		})
	}
}

func TestLoadOverridesFromDir(t *testing.T) {
	testCases := map[string]struct {
		fsLayout   map[string]string
		setupConf  *TargetConfig
		targetDir  string
		wantErr    bool
		wantConfig *TargetConfig
	}{
		"valid overrides": {
			fsLayout: map[string]string{
				"some/dir/containerd-config-path": "custom/config.toml",
				"some/dir/kata-config-path":       "custom/kata.toml",
				"some/dir/restart-systemd-unit":   "false",
				"some/dir/systemd-unit-name":      "custom.service",
			},
			setupConf: &TargetConfig{},
			targetDir: "some/dir",
			wantErr:   false,
			wantConfig: &TargetConfig{
				containerdConfigPath: "custom/config.toml",
				restartSystemdUnit:   false,
				systemdUnitNames:     []string{"custom.service"},
				kataConfigPath:       "custom/kata.toml",
			},
		},
		"partial overrides": {
			fsLayout: map[string]string{
				"some/dir/containerd-config-path": "custom/config.toml",
				"some/dir/restart-systemd-unit":   "true",
			},
			setupConf: &TargetConfig{
				containerdConfigPath: "default/config.toml",
				restartSystemdUnit:   true,
				systemdUnitNames:     []string{"default.service"},
				kataConfigPath:       "default/kata.toml",
			},
			targetDir: "some/dir",
			wantErr:   false,
			wantConfig: &TargetConfig{
				containerdConfigPath: "custom/config.toml",
				restartSystemdUnit:   true,
				systemdUnitNames:     []string{"default.service"},
				kataConfigPath:       "default/kata.toml",
			},
		},
		"no override present": {
			fsLayout: map[string]string{},
			setupConf: &TargetConfig{
				containerdConfigPath: "default/config.toml",
				restartSystemdUnit:   true,
				systemdUnitNames:     []string{"default.service"},
				kataConfigPath:       "default/kata.toml",
			},
			targetDir: "some/dir",
			wantErr:   false,
			wantConfig: &TargetConfig{
				containerdConfigPath: "default/config.toml",
				restartSystemdUnit:   true,
				systemdUnitNames:     []string{"default.service"},
				kataConfigPath:       "default/kata.toml",
			},
		},
		"unit names with multiple separators": {
			fsLayout: map[string]string{
				"some/dir/systemd-unit-name": "unit1, unit2 unit3\nunit4",
			},
			setupConf: &TargetConfig{
				systemdUnitNames: []string{"default.service"},
			},
			targetDir: "some/dir",
			wantErr:   false,
			wantConfig: &TargetConfig{
				systemdUnitNames: []string{"unit1", "unit2", "unit3", "unit4"},
			},
		},
		"containerd-config-path file empty": {
			fsLayout: map[string]string{
				"some/dir/containerd-config-path": "",
			},
			setupConf: &TargetConfig{},
			targetDir: "some/dir",
			wantErr:   true,
		},
		"kata-config-path file empty": {
			fsLayout: map[string]string{
				"some/dir/kata-config-path": "",
			},
			setupConf: &TargetConfig{},
			targetDir: "some/dir",
			wantErr:   true,
		},
		"restart-systemd-unit file empty": {
			fsLayout: map[string]string{
				"some/dir/restart-systemd-unit": "",
			},
			setupConf: &TargetConfig{},
			targetDir: "some/dir",
			wantErr:   true,
		},
		"systemd-unit-name file empty": {
			fsLayout: map[string]string{
				"some/dir/systemd-unit-name": "",
			},
			setupConf: &TargetConfig{},
			targetDir: "some/dir",
			wantErr:   true,
		},
		"unexpected file in target config dir": {
			fsLayout: map[string]string{
				"some/dir/unexpected-file": "unexpected content",
			},
			setupConf: &TargetConfig{},
			targetDir: "some/dir",
			wantErr:   true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			fs := afero.NewMemMapFs()
			for path, content := range tc.fsLayout {
				require.NoError(afero.WriteFile(fs, path, []byte(content), 0o644))
			}

			tc.setupConf.fs = &afero.Afero{Fs: fs}
			err := tc.setupConf.LoadOverridesFromDir(tc.targetDir)
			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			tc.setupConf.fs = nil // Ignore the fs in the comparison
			assert.Equal(tc.wantConfig, tc.setupConf)
		})
	}
}
