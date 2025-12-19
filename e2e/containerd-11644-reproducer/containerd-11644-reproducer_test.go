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

// TestContainerd11644Reproducer is a reproducer and regression test for https://github.com/containerd/containerd/pull/11644.
// In containerd versions <2.0, pulling an image by tag first, then by digest second will cause continerd to re-use the tag-based image name,
// leading to a policy failure.
//
// If the test fails, we either managed to install an old containerd version in our CI cluster
// or containerd regressed upstream on this behavior.
func TestContainerd11644Reproducer(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	deploymentName := "containerd-11644-reproducer"
	runcTester, ccTester := kuberesource.Containerd11644ReproducerTesters(deploymentName)

	// Start the runcTester outside the CC context.
	ct.Init(t, []any{runcTester})
	ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
	t.Cleanup(cancel)
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

	ctx, cancel = context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
	t.Cleanup(cancel)
	err = ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, *ccTester.Name)
	require.NoError(t, err)
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
