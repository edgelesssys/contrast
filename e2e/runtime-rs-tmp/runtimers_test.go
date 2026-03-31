// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package runtimerstmp

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

// TODO: remove when runtime-rs is fully integrated.
// Right now there are some failures left, so we only test that we can start up a container.
// Remove the test and use openssl and other tests when ready.

func TestRuntimeRS(t *testing.T) {
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

	// 'set' currently errors because of wrong measurements, but the debugshell init container should come up.
	require.True(t, t.Run("wait for debugshell", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		defer cancel()
		require.NoError(t, ct.Kubeclient.WaitForContainer(ctx, ct.Namespace, "coordinator-0", "contrast-debug-shell"))
	}), "debugshell start must succeed for subsequent tests")
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()
	os.Exit(m.Run())
}
