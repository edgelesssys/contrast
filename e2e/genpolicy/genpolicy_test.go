// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package genpolicy

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

// TestGenpolicy runs regression tests for generated policies.
func TestGenpolicy(t *testing.T) {
	testCases := kuberesource.GenpolicyRegressionTests()

	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	if platform != platforms.AKSCloudHypervisorSNP {
		t.Skipf("Skipping test for platform %s, only %s is supported. genpolicy is (currently) a regression test for tardev-snapshotter.", platform, platforms.AKSCloudHypervisorSNP)
	}

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	for name, deploy := range testCases {
		t.Run(name, func(t *testing.T) {
			ct := contrasttest.New(t)

			ct.Init(t, kuberesource.PatchRuntimeHandlers([]any{deploy}, runtimeHandler))

			require.True(t, t.Run("generate", func(t *testing.T) {
				require := require.New(t)
				args := []string{
					"--workspace-dir", ct.WorkDir,
					"--reference-values", ct.Platform.String(),
					"--skip-initializer",
					path.Join(ct.WorkDir, "resources.yml"),
				}
				generate := cmd.NewGenerateCmd()
				generate.Flags().String("workspace-dir", "", "") // Make generate aware of root flags
				generate.SetArgs(args)
				generate.SetOut(io.Discard)
				errBuf := &bytes.Buffer{}
				generate.SetErr(errBuf)

				require.NoError(generate.Execute(), "generate failed:\n%s", errBuf.String())
			}), "contrast generate needs to succeed for subsequent tests")

			require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
			t.Cleanup(cancel)
			require.NoError(t, ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, name))
		})
	}
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
