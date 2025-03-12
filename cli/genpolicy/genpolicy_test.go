// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package genpolicy

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scriptTemplate = `#!/bin/sh
set -eu
cd %q

while [ $# -gt 0 ]; do
  case $1 in
    --rego-rules-path=*)
	  printf "%%s" "${1#--rego-rules-path=}" >rules_path
	;;
    --json-settings-path=*)
	  printf "%%s" "${1#--json-settings-path=}" >settings_path
	;;
    --yaml-file=*)
	  printf "%%s" "${1#--yaml-file=}" >yaml_path
	;;
    --runtime-class-names*|--layers-cache-file-path*)
	;;
	*)
	  printf "unknown argument: %%s" "$1" >&2
	  exit 1
	;;
  esac
  shift
done

echo -e "HOME=${HOME}\nXDG_RUNTIME_DIR=${XDG_RUNTIME_DIR}\nDOCKER_CONFIG=${DOCKER_CONFIG}\nREGISTRY_AUTH_FILE=${REGISTRY_AUTH_FILE}" >env_path
`

func TestRunner(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	ctx := context.Background()
	logger := slog.Default()

	t.Setenv("HOME", "/invalid/home")
	t.Setenv("XDG_RUNTIME_DIR", "/invalid/xdg")
	t.Setenv("DOCKER_CONFIG", "/invalid/docker")
	t.Setenv("REGISTRY_AUTH_FILE", "/invalid/registry")

	d := t.TempDir()
	genpolicyBin := []byte(fmt.Sprintf(scriptTemplate, d))

	expectedRulesPath := "/rules.rego"
	rulesPathFile := filepath.Join(d, "rules_path")
	expectedSettingsPath := "/settings.json"
	settingsPathFile := filepath.Join(d, "settings_path")
	cachePath := filepath.Join(d, "cache", "cache.json")
	expectedYAMLPath := filepath.Join(d, "test.yml")
	yamlPathFile := filepath.Join(d, "yaml_path")
	envFile := filepath.Join(d, "env_path")

	r, err := New(expectedRulesPath, expectedSettingsPath, cachePath, genpolicyBin)
	require.NoError(err)

	require.NoError(r.Run(ctx, expectedYAMLPath, nil, logger))

	rulesPath, err := os.ReadFile(rulesPathFile)
	require.NoError(err)
	assert.Equal(expectedRulesPath, string(rulesPath))

	settingsPath, err := os.ReadFile(settingsPathFile)
	require.NoError(err)
	assert.Equal(expectedSettingsPath, string(settingsPath))

	yamlPath, err := os.ReadFile(yamlPathFile)
	require.NoError(err)
	assert.YAMLEq(expectedYAMLPath, string(yamlPath))

	env, err := os.ReadFile(envFile)
	require.NoError(err)
	assert.YAMLEq(expectedYAMLPath, string(yamlPath))
	for _, expected := range []string{
		"HOME=/invalid/home",
		"XDG_RUNTIME_DIR=/invalid/xdg",
		"DOCKER_CONFIG=/invalid/docker",
		"REGISTRY_AUTH_FILE=/invalid/registry",
	} {
		assert.Contains(string(env), expected)
	}

	require.NoError(r.Teardown())
}
