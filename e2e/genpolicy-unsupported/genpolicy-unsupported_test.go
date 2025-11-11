// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package genpolicyunsupported

import (
	"flag"
	"os"
	"path"
	"testing"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

func TestGenpolicyUnsupported(t *testing.T) {
	expectedErrorMessages := map[string]string{
		"volume-directive-no-mount.yml": "The following volumes declared in image config don't have corresponding Kubernetes mounts",
	}

	testdata := "./e2e/genpolicy-unsupported/testdata"
	files, err := os.ReadDir(testdata)
	require.NoError(t, err)

	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.PatchRuntimeHandlers(kuberesource.CoordinatorBundle(), runtimeHandler)

	// required for making everything available for genpolicy
	ct := contrasttest.New(t)
	ct.Init(t, resources)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

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
				err := os.Remove(path.Join(ct.WorkDir, file.Name()))
				if err != nil {
					t.Logf("failed to remove file %q: %s", file.Name(), err)
				}
			})

			err = ct.RunGenerate(t.Context())
			require.Error(err)
			expectedError, ok := expectedErrorMessages[file.Name()]
			if !ok {
				t.Fatalf("test case with file %q does not have expected error message", file.Name())
			}
			require.Contains(err.Error(), expectedError)
		})
	}
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
