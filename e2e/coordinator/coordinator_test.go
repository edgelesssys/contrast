// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package coordinator

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"flag"
	"os"
	"os/exec"
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

	t.Run("signed manifest update", func(t *testing.T) {
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

		// Initial signed manifest update with a valid signature.
		require.NoError(ct.RunSign(ctx, "--out", filepath.Join(ct.WorkDir, "transition.sig")))

		keyBytes, err := os.ReadFile(filepath.Join(ct.WorkDir, "workload-owner.pem"))
		require.NoError(err)
		key, err := manifest.ParseWorkloadOwnerPrivateKey(keyBytes)
		require.NoError(err)
		// Rename the key so that it is not passed to the CLI.
		require.NoError(os.Rename(filepath.Join(ct.WorkDir, "workload-owner.pem"), filepath.Join(ct.WorkDir, "key.pem")))

		require.NoError(ct.RunSet(ctx, "--signature", filepath.Join(ct.WorkDir, "transition.sig")))

		require.True(t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

		// Manifest update without workload owner key or signature should fail.
		require.ErrorContains(ct.RunSet(ctx), "peer not authorized")

		// Signed manifest update with a valid signature computed with Go.
		require.NoError(ct.RunSign(ctx, "--prepare", "--out", filepath.Join(ct.WorkDir, "next-transition")))

		transitionHash, err := os.ReadFile(filepath.Join(ct.WorkDir, "next-transition"))
		require.NoError(err)

		transitionHashShaSum := sha256.Sum256(transitionHash)
		sig, err := ecdsa.SignASN1(rand.Reader, key, transitionHashShaSum[:])
		require.NoError(err)
		require.NoError(os.WriteFile(filepath.Join(ct.WorkDir, "transition.sig"), sig, 0o644))

		require.NoError(ct.RunSet(ctx, "--signature", filepath.Join(ct.WorkDir, "transition.sig")))

		require.True(t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

		// Signed manifest update with a valid signature computed with OpenSSL.
		require.NoError(ct.RunSign(ctx, "--prepare", "--out", filepath.Join(ct.WorkDir, "next-transition")))

		opensslCmd := exec.CommandContext(ctx, "openssl", "dgst", "-sha256", "-sign", filepath.Join(ct.WorkDir, "key.pem"), "-out", filepath.Join(ct.WorkDir, "transition.sig"), filepath.Join(ct.WorkDir, "next-transition"))
		require.NoError(opensslCmd.Run())

		whichCmd := exec.CommandContext(ctx, "which", "openssl")
		out, err := whichCmd.Output()
		require.NoError(err)
		t.Logf("Using OpenSSL from: %s", string(out))

		require.NoError(ct.RunSet(ctx, "--signature", filepath.Join(ct.WorkDir, "transition.sig")))
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
