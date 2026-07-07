// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

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

	// We have two resource sets for testing volumes: VolumeStatefulSet and MySQL. The former is
	// designed for testing, while the latter is a demo artifact we include in releases. The
	// functional tests below mostly use the VolumeStatefulSet, but we include MySQL and some basic
	// checks as a smoke test.
	resources := kuberesource.VolumeStatefulSet()
	resources = append(resources, kuberesource.MySQL()...)

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

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		defer cancel()

		require.NoError(ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, "volume-tester"))
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

		stdOut, stdErr, err = ct.Kubeclient.Exec(ctx, ct.Namespace, pods[0].Name, []string{"cat", filePath})
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
		require.Equal("test\n", stdOut)
	})

	t.Run("file still exists when pod is restarted", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		defer cancel()

		require.NoError(ct.Kubeclient.Restart(ctx, kubeclient.StatefulSet{}, ct.Namespace, "volume-tester"))
		require.NoError(ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, "volume-tester"))

		pods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "volume-tester")
		require.NoError(err)
		require.Len(pods, 1)

		stdOut, stdErr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, pods[0].Name, []string{"cat", filePath})
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
		require.Equal("test\n", stdOut)
	})

	t.Run("MySQL demo works", func(t *testing.T) {
		const (
			mysqlBackend = "mysql-backend"
			mysqlClient  = "mysql-client"
		)

		require := require.New(t)

		// The MySQL server runs initialization on first boot if the volume is empty, which takes
		// a considerable amount of time. Combined with a medium sized image and docker hubs slow
		// pull bandwidth, it can take quite some time until the server comes up.
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(5*time.Minute))
		defer cancel()

		require.NoError(ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, mysqlBackend))
		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, mysqlClient))

		pods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, mysqlClient)
		require.NoError(err)
		require.Len(pods, 1)
		command := []string{"/bin/sh", "-c", `mysql -h 127.137.0.1 -u root -D my_db -e "SELECT * FROM my_table;"`}
		stdout, stderr, err := ct.Kubeclient.ExecRetry(ctx, ct.Namespace, pods[0].Name, mysqlClient, command, time.Second)
		require.NoErrorf(err, "mysql command failed - stderr:\n%s", stderr)
		require.Contains(stdout, "uuid")
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
