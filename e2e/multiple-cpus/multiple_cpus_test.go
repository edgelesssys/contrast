// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package multiplecpus

import (
	"context"
	"flag"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const multiCpu = "multi-cpu"

func TestMultipleCPUs(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.MultiCPU()

	coordinator := kuberesource.CoordinatorBundle()
	resources = append(resources, coordinator...)
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	t.Run("check multiple CPU access", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		require := require.New(t)
		assert := assert.New(t)

		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, multiCpu))

		pods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, multiCpu)
		require.NoError(err)

		// TODO: Check that all requested CPUs are available
		argv := []string{"/bin/sh", "-c", "cat /run/cpu_count"}
		stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, pods[0].Name, argv)
		require.NoError(err)
		require.Empty(stderr)

		cpuCount, err := strconv.Atoi(strings.ReplaceAll(stdout, "\n", ""))
		require.NoError(err)

		assert.Equal(2, cpuCount)
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
