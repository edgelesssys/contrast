// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package genpolicyunsupported

import (
	"flag"
	"log/slog"
	"os"
	"path"
	"testing"

	"github.com/edgelesssys/contrast/cli/genpolicy"
	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

func TestGenpolicyUnsupported(t *testing.T) {
	testdata := "./e2e/genpolicy-unsupported/testdata"
	files, err := os.ReadDir(testdata)
	require.NoError(t, err)
	logger := slog.Default()

	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.PatchRuntimeHandlers(kuberesource.CoordinatorBundle(), runtimeHandler)

	resources = kuberesource.AddPortForwarders(resources)

	// required for making everything available for genpolicy
	ct := contrasttest.New(t)
	ct.Init(t, resources)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			require := require.New(t)
			filePath := path.Join(testdata, file.Name())
			yamlContent, err := os.ReadFile(filePath)
			require.NoError(err)
			require.NoError(os.WriteFile(path.Join(ct.WorkDir, file.Name()), yamlContent, 0o644))
			t.Cleanup(func() {
				require.NoError(os.Remove(path.Join(ct.WorkDir, file.Name())))
			})

			cfg := genpolicy.NewConfig(platform)
			cli, err := genpolicy.New(path.Join(ct.WorkDir, "rules.rego"), path.Join(ct.WorkDir, "settings.json"), path.Join(ct.WorkDir, "layers-cache.json"), cfg.Bin)
			require.NoError(err)
			require.Error(cli.Run(t.Context(), filePath, []string{}, logger))
		})
	}
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
