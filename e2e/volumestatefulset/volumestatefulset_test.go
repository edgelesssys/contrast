// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package volumestatefulset

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

// TestWorkloadSecrets tests that secrets are correctly injected into workloads.
func TestVolumeStatefulSet(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.VolumeStatefulSet()

	coordinator := kuberesource.CoordinatorBundle()

	resources = append(resources, coordinator...)

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	require.True(t, t.Run("deployments become available", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.StatefulSet{}, ct.Namespace, "volume-tester"))
	}), "deployments need to be ready for subsequent tests")

	filePath := "/state/test"
	t.Run("can create file in mounted path", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()

		pods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "volume-tester")
		require.NoError(err)
		require.Len(pods, 1)

		stdOut, stdErr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, pods[0].Name, []string{"sh", "-c", fmt.Sprintf("echo test > %s", filePath)})
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)

		stdOut, stdErr, err = ct.Kubeclient.Exec(ctx, ct.Namespace, pods[0].Name, []string{"sync"})
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)

		stdOut, stdErr, err = ct.Kubeclient.Exec(ctx, ct.Namespace, pods[0].Name, []string{"cat", filePath})
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
		require.Equal("test\n", stdOut)
	})

	t.Run("file still exists when pod is restarted", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(30*time.Second))
		defer cancel()

		require.NoError(ct.Kubeclient.Restart(ctx, kubeclient.StatefulSet{}, ct.Namespace, "volume-tester"))
		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.StatefulSet{}, ct.Namespace, "volume-tester"))

		pods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "volume-tester")
		require.NoError(err)
		require.Len(pods, 1)

		stdOut, stdErr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, pods[0].Name, []string{"cat", filePath})
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
		require.Equal("test\n", stdOut)
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
