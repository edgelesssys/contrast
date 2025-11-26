// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package containerddigestpinning

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestContainerdDigestPinning(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	deploymentName := "containerd-digest-pinning"
	runcTester, ccTester := kuberesource.ContainerdDigestPinningTesters(deploymentName)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)

	// Start the runcTester outside the CC context.
	ct.Init(t, []any{runcTester})
	_, err = ct.Kubeclient.Client.AppsV1().
		Deployments(ct.Namespace).
		Apply(
			ctx,
			runcTester,
			metav1.ApplyOptions{
				FieldManager: "e2e-test",
			},
		)
	require.NoError(t, err)
	err = ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, *runcTester.Name)
	require.NoError(t, err)

	resources := kuberesource.CoordinatorBundle()
	resources = append(resources, ccTester)
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.NoError(t, err)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	err = ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, *ccTester.Name)
	require.NoError(t, err)
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
