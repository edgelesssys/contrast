// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package genpolicy

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const podYAML = `apiVersion: v1
kind: Pod
metadata:
  name: test
`

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
    --config-file=*)
	  printf "%%s" "${1#--config-file=}" >extra_path
	;;
    --runtime-class-names*|--layers-cache-file-path*|--yaml-file*|--base64-out*)
	;;
	*)
	  printf "unknown argument: %%s" "$1" >&2
	  exit 1
	;;
  esac
  shift
done

cat >stdin.yaml

echo -e "HOME=${HOME}\nXDG_RUNTIME_DIR=${XDG_RUNTIME_DIR}\nDOCKER_CONFIG=${DOCKER_CONFIG}\nREGISTRY_AUTH_FILE=${REGISTRY_AUTH_FILE}" >env_path
`

func TestRunner(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	ctx := t.Context()
	logger := slog.Default()

	t.Setenv("HOME", "/invalid/home")
	t.Setenv("XDG_RUNTIME_DIR", "/invalid/xdg")
	t.Setenv("DOCKER_CONFIG", "/invalid/docker")
	t.Setenv("REGISTRY_AUTH_FILE", "/invalid/registry")

	d := t.TempDir()
	genpolicyBin := fmt.Appendf(nil, scriptTemplate, d)

	expectedRulesPath := "/rules.rego"
	rulesPathFile := filepath.Join(d, "rules_path")
	expectedSettingsPath := "/settings.json"
	settingsPathFile := filepath.Join(d, "settings_path")
	expectedExtraPath := "/extra.yml"
	extraPathFile := filepath.Join(d, "extra_path")
	cachePath := filepath.Join(d, "cache", "cache.json")
	envFile := filepath.Join(d, "env_path")

	r, err := New(expectedRulesPath, expectedSettingsPath, cachePath, genpolicyBin)
	require.NoError(err)

	applyConfig, err := kuberesource.UnmarshalApplyConfigurations([]byte(podYAML))
	require.NoError(err)
	require.Len(applyConfig, 1)
	_, err = r.Run(ctx, applyConfig[0], expectedExtraPath, logger)
	require.NoError(err)

	rulesPath, err := os.ReadFile(rulesPathFile)
	require.NoError(err)
	assert.Equal(expectedRulesPath, string(rulesPath))

	settingsPath, err := os.ReadFile(settingsPathFile)
	require.NoError(err)
	assert.Equal(expectedSettingsPath, string(settingsPath))

	extraPath, err := os.ReadFile(extraPathFile)
	require.NoError(err)
	assert.Equal(expectedExtraPath, string(extraPath))

	yamlString, err := os.ReadFile(filepath.Join(d, "stdin.yaml"))
	require.NoError(err)
	assert.YAMLEq(podYAML, string(yamlString))

	env, err := os.ReadFile(envFile)
	require.NoError(err)
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
