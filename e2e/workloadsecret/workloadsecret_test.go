// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package workloadsecret

import (
	"context"
	"encoding/hex"
	"flag"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

// TestWorkloadSecrets tests that secrets are correctly injected into workloads.
func TestWorkloadSecrets(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.Emojivoto(kuberesource.ServiceMeshDisabled)

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

		ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, "vote-bot"))
		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, "emoji"))
		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, "voting"))
		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, "web"))
	}), "deployments need to be ready for subsequent tests")

	// Scale web deployment to 2 replicas.
	require.True(t, t.Run("scale web deployment to 2 pods", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(30*time.Second))
		defer cancel()

		require.NoError(ct.Kubeclient.ScaleDeployment(ctx, ct.Namespace, "web", 2))
		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, "web"))
	}), "web deployment needs to be scaled for subsequent tests")

	var webWorkloadSecretBytes []byte
	var webPods []corev1.Pod
	t.Run("workload secret seed exists", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(30*time.Second))
		defer cancel()

		webPods, err = ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, "web")
		require.NoError(err)
		require.Len(webPods, 2, "pod not found: %s/%s", ct.Namespace, "web")

		stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, webPods[0].Name, []string{"/bin/sh", "-c", "cat /contrast/secrets/workload-secret-seed"})
		require.NoError(err, "stderr: %q", stderr)
		require.NotEmpty(stdout)
		webWorkloadSecretBytes, err = hex.DecodeString(stdout)
		require.NoError(err)
		require.Len(webWorkloadSecretBytes, constants.SecretSeedSize)
	})

	t.Run("workload secret seed is the same between pods in the same deployment", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(30*time.Second))
		defer cancel()

		stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, webPods[1].Name, []string{"/bin/sh", "-c", "cat /contrast/secrets/workload-secret-seed"})
		require.NoError(err, "stderr: %q", stderr)
		require.NotEmpty(stdout)
		otherWebWorkloadSecretBytes, err := hex.DecodeString(stdout)
		require.NoError(err)
		require.Len(otherWebWorkloadSecretBytes, constants.SecretSeedSize)
		require.Equal(webWorkloadSecretBytes, otherWebWorkloadSecretBytes)
	})

	var emojiWorkloadSecretBytes []byte
	t.Run("workload secret seeds differ between deployments by default", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(30*time.Second))
		defer cancel()

		emojiPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, "emoji")
		require.NoError(err)
		require.Len(emojiPods, 1, "pod not found: %s/%s", ct.Namespace, "emoji")

		stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, emojiPods[0].Name, []string{"/bin/sh", "-c", "cat /contrast/secrets/workload-secret-seed"})
		require.NoError(err, "stderr: %q", stderr)
		require.NotEmpty(stdout)
		emojiWorkloadSecretBytes, err = hex.DecodeString(stdout)
		require.NoError(err)
		require.Len(emojiWorkloadSecretBytes, constants.SecretSeedSize)
		require.NotEqual(webWorkloadSecretBytes, emojiWorkloadSecretBytes)
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
