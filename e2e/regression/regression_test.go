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
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kubeapi"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRegression(t *testing.T) {
	yamlDir := "./e2e/regression/testdata/"
	testEntries, err := os.ReadDir(yamlDir)
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

	for _, entry := range testEntries {
		t.Run(entry.Name(), func(t *testing.T) {
			require := require.New(t)

			c := kubeclient.NewForTest(t)

			var resourcesFiles []os.DirEntry
			if entry.IsDir() {
				resourcesFiles, err = os.ReadDir(path.Join(yamlDir, entry.Name()))
				require.NoError(err)
			} else {
				resourcesFiles = append(resourcesFiles, entry)
			}

			var deploymentNames []string
			for _, resourceFile := range resourcesFiles {
				prefixPath := yamlDir
				if entry.IsDir() {
					prefixPath = path.Join(prefixPath, entry.Name())
				}
				yaml, err := os.ReadFile(path.Join(prefixPath, resourceFile.Name()))
				require.NoError(err)
				yaml = bytes.ReplaceAll(yaml, []byte("@@REPLACE_NAMESPACE@@"), []byte(ct.Namespace))

				newResources, err := kuberesource.UnmarshalApplyConfigurations(yaml)
				require.NoError(err)

				// write each resource file to the workdir under its original name
				resourceBytes, err := kuberesource.EncodeResources(newResources...)
				require.NoError(err)
				require.NoError(os.WriteFile(path.Join(ct.WorkDir, resourceFile.Name()), resourceBytes, 0o644))
				t.Cleanup(func() {
					require.NoError(os.Remove(path.Join(ct.WorkDir, resourceFile.Name())))
				})

				// Get deployment name if kind is deployment
				k8sResources, err := kubeapi.UnmarshalUnstructuredK8SResource(yaml)
				require.NoError(err)
				for _, k8sResource := range k8sResources {
					if k8sResource.GetKind() == "Deployment" {
						deploymentName := k8sResource.GetName()
						deploymentNames = append(deploymentNames, deploymentName)
						t.Cleanup(func() {
							t.Log("deleting deployment", deploymentName)
							require.NoError(ct.Kubeclient.Client.AppsV1().Deployments(ct.Namespace).Delete(context.Background(), deploymentName, metav1.DeleteOptions{}))
						})
					}
				}
			}

			// generate, set, deploy and verify the new policy
			require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
			require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
			require.True(t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
			require.True(t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

			ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(3*time.Minute))
			defer cancel()

			for _, deploymentName := range deploymentNames {
				require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, deploymentName))
			}
		})
	}
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
