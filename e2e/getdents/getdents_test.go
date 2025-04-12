// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package getdents

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

const (
	getdent = "getdents-tester"
)

func TestGetDEnts(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	if platform != platforms.AKSCloudHypervisorSNP {
		t.Skipf("Skipping test for platform %s, only %s is supported. getdents is a regression test for tardev-snapshotter.", platform, platforms.AKSCloudHypervisorSNP)
	}

	resources := kuberesource.GetDEnts()

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

	ct.Init(t, resources)

	// Call generate to patch the runtime class.
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

		require.NoError(generate.Execute())
	}), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	// This tests an upstream bug in TarFS where a 'getdents' call hangs in an infinite loop when too many files are in a directory.
	// If the 'find' command results in a timeout, the test fails.
	t.Run("call find on large folder", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(30*time.Second))
		defer cancel()

		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, getdent))

		pods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, getdent)
		require.NoError(err)
		require.Len(pods, 1, "pod not found: %s/%s", ct.Namespace, getdent)

		stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, pods[0].Name, []string{"/bin/sh", "-c", "find /toomany | wc -l"})
		require.NoError(err, "stderr: %q", stderr)
		require.Equal("10001\n", stdout, "output: %s", stdout)
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
