// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package gpu

import (
	"bytes"
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

const (
	gpuPodName = "gpu-pod"
	gpuName    = "NVIDIA H100 PCIe"
)

// TestGPU runs e2e tests on an GPU-enabled Contrast.
func TestGPU(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.OpenSSL()
	coordinator := kuberesource.CoordinatorBundle()

	resources = append(resources, coordinator...)

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	applyGPUPod := func(t *testing.T) {
		yaml, err := os.ReadFile("./e2e/gpu/testdata/gpu-pod.yaml")
		require.NoError(t, err)

		yaml = bytes.ReplaceAll(
			bytes.ReplaceAll(yaml, []byte("@@REPLACE_NAMESPACE@@"), []byte(ct.Namespace)),
			[]byte("@@REPLACE_RUNTIME@@"), []byte(ct.RuntimeClassName),
		)

		ct.ApplyFromYAML(t, yaml)
	}

	require.True(t, t.Run("apply GPU pod", applyGPUPod), "GPU pod needs to deploy successfully for subsequent tests")

	t.Run("check GPU availability", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(5*time.Minute))
		defer cancel()

		require := require.New(t)

		err := ct.Kubeclient.WaitForPod(ctx, ct.Namespace, gpuPodName)
		require.NoError(err, "GPU pod %s did not start", gpuPodName)

		argv := []string{"/bin/sh", "-c", "nvidia-smi"}
		stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, gpuPodName, argv)
		require.NoError(err, "stderr: %q", stderr)

		require.Contains(stdout, gpuName, "nvidia-smi output should contain %s", gpuName)
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
