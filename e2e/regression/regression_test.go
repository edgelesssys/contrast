// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package regression

import (
	"bytes"
	"flag"
	"os"
	"testing"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

var (
	imageReplacementsFile, namespaceFile string
	skipUndeploy                         bool
)

func TestRegression(t *testing.T) {
	require := require.New(t)

	ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, skipUndeploy)

	resources := kuberesource.CoordinatorBundle()

	yaml, err := os.ReadFile("./e2e/regression/test-data/redis-alpine.yaml")
	require.NoError(err)
	yaml = bytes.ReplaceAll(yaml, []byte("REPLACE_NAMESPACE"), []byte(ct.Namespace))
	yaml = bytes.ReplaceAll(yaml, []byte("REPLACE_RUNTIME"), []byte(kuberesource.RuntimeHandler))
	yamlResources, err := kuberesource.UnmarshalApplyConfigurations(yaml)
	require.NoError(err)
	resources = append(resources, yamlResources...)

	resources = kuberesource.AddPortForwarders(resources)
	t.Logf("config:\n%s", resources)

	ct.Init(t, resources)

	require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.BoolVar(&skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}
