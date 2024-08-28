// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

///go:build e2e

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

var (
	imageReplacementsFile, namespaceFile, platformStr string
	_skipUndeploy                                     bool // just here for interoptability, ignored in this test
)

func TestRegression(t *testing.T) {
	yamlDir := "./e2e/regression/testdata/"
	files, err := os.ReadDir(yamlDir)
	require.NoError(t, err)

	platform, err := platforms.FromString(platformStr)
	require.NoError(t, err)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, platform, false)

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
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			yaml, err := os.ReadFile(yamlDir + file.Name())
			require.NoError(err)
			yaml = bytes.ReplaceAll(yaml, []byte("@@REPLACE_NAMESPACE@@"), []byte(ct.Namespace))

			newResources, err := kuberesource.UnmarshalApplyConfigurations(yaml)
			require.NoError(err)

			newResources = kuberesource.PatchRuntimeHandlers(newResources, runtimeHandler)
			newResources = kuberesource.AddPortForwarders(newResources)

			// write the new resources.yaml
			resourceBytes, err := kuberesource.EncodeResources(newResources...)
			require.NoError(err)
			require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yaml"), resourceBytes, 0o644))

			deploymentName, _ := strings.CutSuffix(file.Name(), ".yaml")

			t.Cleanup(func() {
				// delete the deployment
				require.NoError(ct.Kubeclient.Client.AppsV1().Deployments(ct.Namespace).Delete(ctx, deploymentName, metav1.DeleteOptions{}))
			})

			// generate, set, deploy and verify the new policy
			require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
			require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
			require.True(t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
			require.True(t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

			require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, deploymentName))
		})
	}
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.StringVar(&platformStr, "platform", "", "Deployment platform")

	// ignored and just here for interoptability, we always undeploy to save resources
	flag.BoolVar(&_skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}
