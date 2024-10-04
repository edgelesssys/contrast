// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

///go:build e2e

package aksruntime

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	imageReplacementsFile, namespaceFile, platformStr string
	skipUndeploy                                      bool
)

func TestAKSRuntime(t *testing.T) {
	// TODO: Log kata information

	require := require.New(t)

	platform, err := platforms.FromString(platformStr)
	require.NoError(err)
	c := kubeclient.NewForTest(t)

	resources := kuberesource.OpenSSL()
	resources = kuberesource.PatchRuntimeHandlers(resources, "kata-isolation-cc")

	namespace, err := os.ReadFile(namespaceFile)
	require.NoError(err)
	deploymentsClient := c.Client.AppsV1().Deployments(string(namespace))

	deploymentsClient.Create(context.TODO(), resources, metav1.CreateOptions{})
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.StringVar(&platformStr, "platform", "", "Deployment platform")
	flag.BoolVar(&skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}
