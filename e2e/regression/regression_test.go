// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package regression

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRegression(t *testing.T) {
	yamlDir := "./e2e/regression/testdata/"
	files, err := os.ReadDir(yamlDir)
	require.NoError(t, err)

	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	ct := contrasttest.New(t)

	// Initially just deploy the coordinator bundle

	resources := kuberesource.CoordinatorBundle()
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	for _, file := range files {
		t.Run(file.Name(), func(t *testing.T) {
			require := require.New(t)

			c := kubeclient.NewForTest(t)

			yaml, err := os.ReadFile(yamlDir + file.Name())
			require.NoError(err)
			yaml = bytes.ReplaceAll(yaml, []byte("@@REPLACE_NAMESPACE@@"), []byte(ct.Namespace))

			newResources, err := kuberesource.UnmarshalApplyConfigurations(yaml)
			require.NoError(err)

			newResources = kuberesource.PatchRuntimeHandlers(newResources, runtimeHandler)
			newResources = kuberesource.AddPortForwarders(newResources)

			// write the new resources.yml
			resourceBytes, err := kuberesource.EncodeResources(append(resources, newResources...)...)
			require.NoError(err)
			require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yml"), resourceBytes, 0o644))

			deploymentName, _ := strings.CutSuffix(file.Name(), ".yml")

			t.Cleanup(func() {
				// delete the deployment
				require.NoError(ct.Kubeclient.Client.AppsV1().Deployments(ct.Namespace).Delete(context.Background(), deploymentName, metav1.DeleteOptions{}))
			})

			// generate, set, deploy and verify the new policy
			require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
			require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
			require.True(t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
			require.True(t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

			ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(3*time.Minute))
			defer cancel()
			require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, deploymentName))
		})
	}
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
