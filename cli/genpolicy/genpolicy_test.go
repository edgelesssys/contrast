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
`

func TestRunner(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	ctx := context.Background()
	logger := slog.Default()

	d := t.TempDir()
	genpolicyBin := []byte(fmt.Sprintf(scriptTemplate, d))

	expectedRulesPath := "/rules.rego"
	rulesPathFile := filepath.Join(d, "rules_path")
	expectedSettingsPath := "/settings.json"
	settingsPathFile := filepath.Join(d, "settings_path")
	cachePath := filepath.Join(d, "cache", "cache.json")
	expectedYAMLPath := filepath.Join(d, "test.yml")
	yamlPathFile := filepath.Join(d, "yaml_path")

	r, err := New(expectedRulesPath, expectedSettingsPath, cachePath, genpolicyBin)
	require.NoError(err)

	require.NoError(r.Run(ctx, expectedYAMLPath, logger))

	rulesPath, err := os.ReadFile(rulesPathFile)
	require.NoError(err)
	assert.Equal(expectedRulesPath, string(rulesPath))

	settingsPath, err := os.ReadFile(settingsPathFile)
	require.NoError(err)
	assert.Equal(expectedSettingsPath, string(settingsPath))

	yamlPath, err := os.ReadFile(yamlPathFile)
	require.NoError(err)
	assert.YAMLEq(expectedYAMLPath, string(yamlPath))

	require.NoError(r.Teardown())
}
