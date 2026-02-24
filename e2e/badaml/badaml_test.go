// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package badaml

import (
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

var expectSuccessfulAttack = false

func TestBadAML(t *testing.T) {
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
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

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

	require.True(t, t.Run("check attack success", func(t *testing.T) {
		if expectSuccessfulAttack {
			require.Equal(t, "cafebabe", content, "the content of /run/deadbeef.bin should be 'cafebabe' if the attack was successful")
		} else {
			require.Equal(t, "deadbeef", content, "the content of /run/deadbeef.bin should be 'deadbeef' if the attack was not successful")
		}
	}))
}

func TestMain(m *testing.M) {
	flag.BoolVar(&expectSuccessfulAttack, "expect-successful-attack", false, "if true, check the BadAML attack was successful")
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
