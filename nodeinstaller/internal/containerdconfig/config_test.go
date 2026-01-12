// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package containerdconfig

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/pelletier/go-toml/v2"
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

// Legacy test with golden values to ensure the config manipulation stays consistent over time.
// For new tests, prefer to add new tests that test specific functions instead.
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

func TestConfig(t *testing.T) {
	t.Run("FromPath", func(t *testing.T) {
		testCases := map[string]struct {
			prepareFS  func(tmpDir string) error
			path       string
			wantConfig Config
			wantErr    bool
		}{
			"ok": {
				prepareFS: func(tmpDir string) error {
					cfg := config{Version: 2}
					data, err := toml.Marshal(cfg)
					if err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(tmpDir, "config.toml"), data, 0o644)
				},
				path: "config.toml",
				wantConfig: Config{
					path:   "@tmpDir@/config.toml",
					config: config{Version: 2},
				},
			},
			"template": {
				prepareFS: func(tmpDir string) error {
					cfg := config{Version: 2}
					data, err := toml.Marshal(cfg)
					if err != nil {
						return err
					}
					if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), data, 0o644); err != nil {
						return err
					}
					// The template file is only read as bytes, it may contain invalid TOML.
					return os.WriteFile(filepath.Join(tmpDir, "config.toml.tmpl"), []byte("foo"), 0o644)
				},
				path: "config.toml.tmpl",
				wantConfig: Config{
					path:   "@tmpDir@/config.toml.tmpl",
					config: config{Version: 2},
				},
			},
			"template empty": {
				prepareFS: func(tmpDir string) error {
					cfg := config{Version: 2}
					data, err := toml.Marshal(cfg)
					if err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(tmpDir, "config.toml"), data, 0o644)
				},
				path: "config.toml.tmpl",
				wantConfig: Config{
					path:   "@tmpDir@/config.toml.tmpl",
					config: config{Version: 2},
				},
			},
			"config doesn't exist": {
				wantErr: true,
				path:    "config.toml",
			},
			"content invalid": {
				prepareFS: func(tmpDir string) error {
					return os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte("invalid-toml"), 0o644)
				},
				path:    "config.toml",
				wantErr: true,
			},
		}
		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				assert := assert.New(t)
				require := require.New(t)

				tmpDir := t.TempDir()
				if tc.prepareFS != nil {
					require.NoError(tc.prepareFS(tmpDir))
				}

				gotConfig, err := FromPath(filepath.Join(tmpDir, tc.path))
				if tc.wantErr {
					require.Error(err)
					return
				}
				require.NoError(err)
				assert.Equal(strings.ReplaceAll(tc.wantConfig.path, "@tmpDir@", tmpDir), gotConfig.path)
				assert.Equal(tc.wantConfig.config, gotConfig.config)
			})
		}
	})
	t.Run("AddRuntime", func(t *testing.T) {
		assert := assert.New(t)

		cfg := Config{
			config: config{Version: 2},
		}
		runtimeName := "my-runtime"
		runtime := Runtime{
			Type: "io.containerd.runc.v2",
			Path: "/opt/my-runtime/bin/containerd-runtime",
		}
		cfg.AddRuntime(runtimeName, runtime)

		expected := Config{
			config: config{
				Version: 2,
				Plugins: map[string]any{
					criFQDN(2): map[string]any{
						"containerd": map[string]any{
							"runtimes": map[string]any{
								runtimeName: runtime,
							},
						},
					},
				},
			},
		}
		assert.Equal(expected.config, cfg.config)
	})
	t.Run("EnableDebug", func(t *testing.T) {
		assert := assert.New(t)

		cfg := Config{
			config: config{Version: 2},
		}
		cfg.EnableDebug()

		expected := Config{
			config: config{
				Version: 2,
				Debug: debug{
					Level: "debug",
				},
			},
		}
		assert.Equal(expected.config, cfg.config)
	})
	t.Run("Write", func(t *testing.T) {
		testCases := map[string]struct {
			prepareFS  func(tmpDir string) error
			config     Config
			wantFile   string
			wantBackup bool
			wantErr    bool
		}{
			"new": {
				config: Config{
					config: config{Version: 2},
					path:   "config.toml",
				},
				wantFile:   "config.toml",
				wantBackup: false,
			},
			"update existing": {
				prepareFS: func(tmpDir string) error {
					cfg := config{Version: 2}
					data, err := toml.Marshal(cfg)
					if err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(tmpDir, "config.toml"), data, 0o644)
				},
				config: Config{
					raw:    []byte("foobar"),
					config: config{Version: 3},
					path:   "config.toml",
				},
				wantFile:   "config.toml",
				wantBackup: true,
			},
			"no changes no update": {
				prepareFS: func(tmpDir string) error {
					cfg := config{Version: 2}
					data, err := toml.Marshal(cfg)
					if err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(tmpDir, "config.toml"), data, 0o644)
				},
				config: Config{
					config: config{Version: 2},
					raw:    requireMarshalTOML(t, config{Version: 2}),
					path:   "config.toml",
				},
				wantFile:   "config.toml",
				wantBackup: false,
			},
			"only template exists": {
				prepareFS: func(tmpDir string) error {
					cfg := config{Version: 2}
					data, err := toml.Marshal(cfg)
					if err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(tmpDir, "config.toml.tmpl"), data, 0o644)
				},
				config: Config{
					path:   "config.toml.tmpl",
					config: config{Version: 2},
				},
				wantFile:   "config.toml.tmpl",
				wantBackup: false,
			},
		}
		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				assert := assert.New(t)
				require := require.New(t)

				tmpDir := t.TempDir()
				tc.config.path = filepath.Join(tmpDir, tc.config.path)
				tc.wantFile = filepath.Join(tmpDir, tc.wantFile)

				if tc.prepareFS != nil {
					require.NoError(tc.prepareFS(tmpDir))
				}

				err := tc.config.Write()
				if tc.wantErr {
					require.Error(err)
					return
				}
				require.NoError(err)

				// Check wantFile exists.
				assert.FileExists(tc.wantFile)
				// Check content.
				inFile, err := os.ReadFile(tc.wantFile)
				require.NoError(err)
				var inFileCfg config
				require.NoError(toml.Unmarshal(inFile, &inFileCfg))
				assert.Equal(tc.config.config, inFileCfg)
				// Check backup exists.
				bakExists := fileWithSuffixExists(t, tmpDir, ".bak")
				if tc.wantBackup {
					assert.True(bakExists, "expected backup to exist")
				} else {
					assert.False(bakExists, "expected no backup to exist")
				}
			})
		}
	})
}

func fileWithSuffixExists(t *testing.T, path, suffix string) bool {
	t.Helper()
	files, err := os.ReadDir(path)
	require.NoError(t, err)
	for _, f := range files {
		if strings.HasSuffix(f.Name(), suffix) && f.Type().IsRegular() {
			return true
		}
	}
	return false
}

func requireMarshalTOML(t *testing.T, v any) []byte {
	t.Helper()
	data, err := toml.Marshal(v)
	require.NoError(t, err)
	return data
}
