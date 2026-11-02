// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// runs-on: Metal-QEMU-SNP,Metal-QEMU-SNP-GPU,Metal-QEMU-TDX,Metal-QEMU-TDX-GPU
// test-if: nix:kata.systemd-units
// test-if: nix:contrast.node-installer-image.gpu.os-image

//go:build e2e

package systemdunits

import (
	"context"
	"flag"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

const (
	pod       = "coordinator-0"
	container = "contrast-debug-shell"
)

// TestSystemdUnits checks that the podvm only starts units that are expected to be startable.
func TestSystemdUnits(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	require.True(t, contrasttest.Flags.InsecureEnableDebugShell,
		"the --insecure-enable-debug-shell-access flag must be set to true to inspect the guest")

	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)
	resources := kuberesource.CoordinatorBundle()
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)
	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("wait for debugshell", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		defer cancel()
		require.NoError(t, ct.Kubeclient.WaitForContainer(ctx, ct.Namespace, pod, container))
	}), "debugshell start must succeed for subsequent tests")

	var observed, external []string
	require.True(t, t.Run("collect started units", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()
		guest := func(command string) string {
			argv := []string{"debugshell", command}
			stdout, stderr, err := ct.Kubeclient.ExecContainer(ctx, ct.Namespace, pod, container, argv)
			require.NoError(t, err, "running %q:\nstdout:\n%s\nstderr:\n%s", command, stdout, stderr)
			return stdout
		}

		journalUnits, err := parseJournal(guest(journalCommand))
		require.NoError(t, err)
		// If the journal of the initrd didn't survive the switch to the final system, we would silently miss all units of the initrd.
		require.Contains(t, journalUnits, "initrd.target", "journal doesn't cover the initrd")
		require.Contains(t, journalUnits, "kata-containers.target", "journal doesn't cover the final system")

		started, ext := parseShow(guest(showCommand))
		observed, external = append(journalUnits, started...), ext
	}), "collecting the started units needs to succeed for subsequent tests")

	t.Run("only expected units are started", func(t *testing.T) {
		golden := expectedUnits
		if platforms.IsGPU(platform) {
			golden = expectedUnitsGPU
		}
		unexpected, skipped := checkUnits(observed, parseExpected(golden))
		t.Logf("checked %d distinct started units", len(slices.Compact(slices.Sorted(slices.Values(observed)))))
		t.Logf("skipped units that can't execute anything: %s", strings.Join(skipped, " "))
		t.Logf("skipped units not started by systemd: %s", strings.Join(external, " "))
		require.Empty(t, unexpected, "the podvm started units that kata.systemd-units doesn't consider startable")
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()
	os.Exit(m.Run())
}
