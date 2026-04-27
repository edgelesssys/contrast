// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package insecure

import (
	"context"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	secureDeployment   = "secure-pod"
	insecureDeployment = "insecure-pod"
)

// TestInsecure deploys a secure and an insecure pod side by side and verifies
// that only the secure pod runs inside a TEE.
func TestInsecure(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)

	insecurePlatform := platform.InsecureVariant()
	if insecurePlatform == platforms.Unknown {
		t.Skip("no insecure variant for platform", platform)
	}

	// The generate and verify commands require this env var for insecure platforms.
	t.Setenv("CONTRAST_ALLOW_INSECURE_RUNTIMES", "1")

	ct := contrasttest.New(t)
	ct.Platform = insecurePlatform // Required so RunGenerate/RunVerify pass --INSECURE.

	secureHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)
	insecureHandler, err := manifest.RuntimeHandler(insecurePlatform)
	require.NoError(t, err)

	resources := kuberesource.CoordinatorBundle()
	// Patch the coordinator with the insecure runtime handler.
	resources = kuberesource.PatchRuntimeHandlers(resources, insecureHandler)
	resources = kuberesource.AddPortForwarders(resources)

	// Add deployments *after* PatchRuntimeHandlers to retain control over the RuntimeClassNames.
	resources = append(resources, kuberesource.DeploymentWithRuntimeClass(secureDeployment, secureHandler))
	resources = append(resources, kuberesource.DeploymentWithRuntimeClass(insecureDeployment, insecureHandler))

	ct.Init(t, resources)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	t.Run("pods use correct runtime classes", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		defer cancel()
		require := require.New(t)

		securePods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, secureDeployment)
		require.NoError(err)
		require.Len(securePods, 1)
		assert.True(t, strings.HasPrefix(*securePods[0].Spec.RuntimeClassName, secureHandler))

		insecurePods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, insecureDeployment)
		require.NoError(err)
		require.Len(insecurePods, 1)
		assert.True(t, strings.HasPrefix(*insecurePods[0].Spec.RuntimeClassName, insecureHandler))
	})

	t.Run("pods start", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		defer cancel()
		require := require.New(t)

		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, secureDeployment))
		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, insecureDeployment))
	})

	t.Run("secure pod runs in TEE", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()
		require := require.New(t)

		stdout, stderr, err := ct.Kubeclient.ExecDeployment(ctx, ct.Namespace, secureDeployment, []string{
			"/usr/local/bin/bash", "-c", "dmesg | grep -i -E 'tdx|sev|snp'",
		})
		require.NoError(err, "stderr: %q", stderr)
		require.NotEmpty(strings.TrimSpace(stdout), "expected TEE-related dmesg output in secure pod")
	})

	t.Run("insecure pod does not run in TEE", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		// grep exits with 1 when no lines match, so we expect an error here.
		stdout, _, _ := ct.Kubeclient.ExecDeployment(ctx, ct.Namespace, insecureDeployment, []string{
			"/usr/local/bin/bash", "-c", "dmesg | grep -i -E 'tdx|sev|snp'",
		})
		assert.Empty(t, strings.TrimSpace(stdout), "expected no TEE-related dmesg output in insecure pod")
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
