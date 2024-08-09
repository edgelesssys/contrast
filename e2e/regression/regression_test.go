// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package regression

import (
	"bytes"
	"context"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

var (
	imageReplacementsFile, namespaceFile string
	_skipUndeploy                        bool // just here for interoptability, ignored in this test
)

func TestRegression(t *testing.T) {
	yamlDir := "./e2e/regression/test-data/"
	files, err := os.ReadDir(yamlDir)
	require.NoError(t, err)

	// TODO(miampf): Make this configurable
	platform := platforms.AKSCloudHypervisorSNP

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	for _, file := range files {
		t.Run(file.Name(), func(t *testing.T) {
			require := require.New(t)

			c := kubeclient.NewForTest(t)
			ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, false)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			resources := kuberesource.CoordinatorBundle()

			yaml, err := os.ReadFile(yamlDir + file.Name())
			require.NoError(err)
			yaml = bytes.ReplaceAll(yaml, []byte("REPLACE_NAMESPACE"), []byte(ct.Namespace))

			yamlResources, err := kuberesource.UnmarshalApplyConfigurations(yaml)
			require.NoError(err)
			resources = append(resources, yamlResources...)

			resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

			resources = kuberesource.AddPortForwarders(resources)

			ct.Init(t, resources)

			require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
			require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
			require.True(t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
			require.True(t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

			deploymentName, _ := strings.CutSuffix(file.Name(), ".yaml")
			require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, deploymentName))
		})
	}
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")

	// ignored and just here for interoptability, we always undeploy to save resources
	flag.BoolVar(&_skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}
