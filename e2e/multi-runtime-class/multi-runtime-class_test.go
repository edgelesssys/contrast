// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package multiruntimeclass

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
	defaultDeployment    = "default-runtime-class"
	overriddenDeployment = "overridden-runtime-class"
)

func TestMultiRuntimeClass(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	// Only run on GPU hosts, where we can meaningfully test two runtime classes
	if !platforms.IsGPU(platform) {
		return
	}
	ct := contrasttest.New(t)

	defaultRuntimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)
	nonGPURuntimeHandler := strings.ReplaceAll(defaultRuntimeHandler, "-gpu", "")

	resources := kuberesource.CoordinatorBundle()
	resources = kuberesource.PatchRuntimeHandlers(resources, defaultRuntimeHandler)
	resources = kuberesource.AddPortForwarders(resources)

	// Adding these resources *after* PatchRuntimeHandlers to retain control over the RuntimeClassNames.
	resources = append(resources, kuberesource.DeploymentWithRuntimeClass(defaultDeployment, defaultRuntimeHandler))
	resources = append(resources, kuberesource.DeploymentWithRuntimeClass(overriddenDeployment, nonGPURuntimeHandler))

	ct.Init(t, resources)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	t.Run("the pods use the correct runtime-class", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		defer cancel()
		require := require.New(t)
		assert := assert.New(t)

		defaultPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, defaultDeployment)
		require.NoError(err)
		require.Len(defaultPods, 1)
		actualDefaultRuntimeClass := *defaultPods[0].Spec.RuntimeClassName
		assert.Equal(defaultRuntimeHandler, actualDefaultRuntimeClass[0:len(defaultRuntimeHandler)])

		overriddenPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, overriddenDeployment)
		require.NoError(err)
		require.Len(overriddenPods, 1)
		actualOverriddenRuntimeClass := *overriddenPods[0].Spec.RuntimeClassName
		assert.Equal(nonGPURuntimeHandler, actualOverriddenRuntimeClass[0:len(nonGPURuntimeHandler)])
	})

	t.Run("the pods using each runtime-class start", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		defer cancel()
		require := require.New(t)

		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, defaultDeployment))
		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, overriddenDeployment))
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
