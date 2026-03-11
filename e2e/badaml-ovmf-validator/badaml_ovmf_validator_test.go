// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package badamlovmfvalidator

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

const coordinatorPod = "coordinator-0"

func TestBadAMLOVMFValidator(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)

	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.CoordinatorBundle()
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)
	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	t.Run("check VM fails to start due to OVMF ACPI validation", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(5*time.Minute))
		defer cancel()

		c := kubeclient.NewForTest(t)

		t.Log("Waiting for pod sandbox creation to fail (OVMF should reject injected ACPI table)")
		err := c.WaitForEvent(ctx, kubeclient.StartingBlocked, kubeclient.Pod{}, ct.Namespace, coordinatorPod)
		require.NoError(err, "expected pod to fail starting due to OVMF ACPI validation rejecting the injected SSDT")
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()
	os.Exit(m.Run())
}
