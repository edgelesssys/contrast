// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package badamlvuln

import (
	"context"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

// BadAMLTest is the shared logic between the badaml-vuln and badaml-sandbox tests.
func BadAMLTest(t *testing.T, expectSuccessfulAttack bool) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)

	ct := contrasttest.New(t)

	require.True(t, contrasttest.Flags.InsecureEnableDebugShell, "the --insecure-enable-debug-shell-access flag must be set to true to extract the initrd start address")

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)
	resources := kuberesource.CoordinatorBundle()
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)
	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	if platform == platforms.MetalQEMUTDX {
		// TODO(katexochen): Set on TDX currently fails, as we are still measuring the ACPI tables, so
		// the injected BadAML table is detected during remote attestation. For TDX-GPU, we don't measure
		// the table so the attack works.
		require.True(t, t.Run("wait for debugshell", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
			defer cancel()
			require.NoError(t, ct.Kubeclient.WaitForContainer(ctx, ct.Namespace, "coordinator-0", "contrast-debug-shell"))
		}), "debugshell start must succeed for subsequent tests")
	} else {
		require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	}

	var content string
	require.True(t, t.Run("get content of /run/deadbeef.bin", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()
		cmd := []string{"debugshell", `hexdump -e '1/1 "%02x"' -n4 /run/deadbeef.bin`}
		stdout, stderr, err := ct.Kubeclient.ExecContainer(ctx, ct.Namespace, "coordinator-0", "contrast-debug-shell", cmd)
		require.NoError(t, err, "running %q:\nstdout:\n%s\nstderr:\n%s", cmd, stdout, stderr)
		content = stdout
	}), "getting content of /run/deadbeef.bin needs to succeed for subsequent tests")

	require.True(t, t.Run("get dmesg logs", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()
		cmd := []string{"debugshell", "dmesg | grep -i acpi"}
		stdout, stderr, err := ct.Kubeclient.ExecContainer(ctx, ct.Namespace, "coordinator-0", "contrast-debug-shell", cmd)
		require.NoError(t, err, "running %q:\nstdout:\n%s\nstderr:\n%s", cmd, stdout, stderr)
		t.Log(stdout)
	}))

	name := "check attack is"
	if !expectSuccessfulAttack {
		name += " not"
	}
	name += " successful"
	require.True(t, t.Run(name, func(t *testing.T) {
		if expectSuccessfulAttack {
			require.Equal(t, "cafebabe", content, "the content of /run/deadbeef.bin should be 'cafebabe' if the attack was successful")
		} else {
			require.Equal(t, "deadbeef", content, "the content of /run/deadbeef.bin should be 'deadbeef' if the attack was not successful")
		}
	}))
}
