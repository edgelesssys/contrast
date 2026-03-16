// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package coordinator

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

func TestCoordinator(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	t.Run("recover configmap store", func(t *testing.T) {
		require := require.New(t)
		ct := contrasttest.New(t)

		resources := kuberesource.CoordinatorBundle()
		resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
		resources = kuberesource.AddPortForwarders(resources)
		ct.Init(t, resources)

		require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
		require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
		require.True(t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
		require.True(t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		t.Cleanup(cancel)

		unstructured, err := kuberesource.ResourcesToUnstructured(resources)
		require.NoError(err)

		// Delete whole statefulset so the owned configmap store is also deleted.
		require.NoError(ct.Kubeclient.Delete(ctx, unstructured...))
		// Wait until all objects are deleted before applying the resources again,
		// otherwise some resources may be deleted after being reapplied.
		require.NoError(ct.Kubeclient.WaitForDeletion(ctx, unstructured...))
		require.True(t.Run("apply resources after deleting them", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

		require.NoError(ct.Kubeclient.WaitForCoordinator(ctx, ct.Namespace))

		require.ErrorContains(ct.RunRecover(ctx), "no state to recover from")

		historyBytes, err := os.ReadFile(filepath.Join(ct.WorkDir, "verify/history.yml"))
		require.NoError(err)

		applyConfigs, err := kuberesource.UnmarshalApplyConfigurations(historyBytes)
		require.NoError(err)

		applyConfigs = kuberesource.PatchNamespaces(applyConfigs, ct.Namespace)

		historyUnstructured, err := kuberesource.ResourcesToUnstructured(applyConfigs)
		require.NoError(err)
		require.NoError(ct.Kubeclient.Apply(ctx, historyUnstructured...))

		t.Run("recover after restoring configmaps", ct.Recover)
	})

	t.Run("atomic manifest update", func(t *testing.T) {
		require := require.New(t)
		ct := contrasttest.New(t)

		resources := kuberesource.CoordinatorBundle()
		resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
		resources = kuberesource.AddPortForwarders(resources)
		ct.Init(t, resources)

		require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
		require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		t.Cleanup(cancel)

		// Initial atomic set manifest.
		require.NoError(ct.RunSet(ctx, "--atomic"))

		require.True(t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

		require.NoError(ct.RunSet(ctx, "--atomic"))

		// Second set manifest does not have the correct latest transition hash.
		require.ErrorContains(ct.RunSet(ctx, "--atomic"), "does not match latest state")

		// Normal set manifest should still work.
		require.NoError(ct.RunSet(ctx))

		require.ErrorContains(ct.RunSet(ctx, "--atomic", "--latest-transition", "0123456789abcdef"), "does not match latest state")

		require.NoError(ct.RunVerify(ctx))
		transitionHash, err := os.ReadFile(filepath.Join(ct.WorkDir, "verify/latest-transition"))
		require.NoError(err)
		require.NoError(ct.RunSet(ctx, "--atomic", "--latest-transition", string(transitionHash)))
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
