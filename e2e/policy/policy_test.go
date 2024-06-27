// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

///go:build e2e

package policy

import (
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

func TestPolicy(t *testing.T) {
	ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, skipUndeploy)

	resources := kuberesource.CoordinatorBundle()

	pod := kuberesource.Deployment("test-deployment", ct.Namespace).
		WithLabels(map[string]string{
			"app.kubernetes.io/name": "hello-world",
		}).
		WithSpec(kuberesource.DeploymentSpec().
			WithReplicas(1).
			WithSelector(kuberesource.LabelSelector().
				WithMatchLabels(map[string]string{
					"app.kubernetes.io/name": "hello-world",
				}),
			).
			WithTemplate(kuberesource.PodTemplateSpec().
				WithLabels(map[string]string{
					"app.kubernetes.io/name": "hello-world",
				}).
				WithSpec(kuberesource.PodSpec().
					WithContainers(
						kuberesource.Container().
							WithName("hello-world").
							WithImage("hello-world:latest"),
					),
				),
			),
		)

	resources = append(resources, pod)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	// initial deployment with pod allowed
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	t.Run("pod cannot join after it was removed from the manifest", func(t *testing.T) {
		// TODO: Remove the policy hash for the test-deployment from `manifest.json` and set the new manifest.
		// (look at `openssl_test.go` for an example of editing the manifest file)
	})

	t.Run("manifest does not allow pod with valid policy", func(t *testing.T) {
		// TODO: Create a new pod with a valid policy but with contents that don't match the manifest (like ports?).
		// Joining of that pod should fail.
	})
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.BoolVar(&skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}
