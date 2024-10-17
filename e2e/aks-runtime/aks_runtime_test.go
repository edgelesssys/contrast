// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package aksruntime

import (
	"context"
	"flag"
	"os"
	"path"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/az"
	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kubeapi"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testContainer = "testcontainer"

var (
	imageReplacementsFile, namespaceFile, _platformStr string
	skipUndeploy                                       bool
)

func TestAKSRuntime(t *testing.T) {
	require := require.New(t)

	workdir := t.TempDir()

	f, err := os.Open(imageReplacementsFile)
	require.NoError(err)
	imageReplacements, err := kuberesource.ImageReplacementsFromFile(f)
	require.NoError(err)
	namespace := contrasttest.MakeNamespace(t)

	// Log versions
	kataPolicyGenV, err := az.KataPolicyGenVersion()
	require.NoError(err)
	rg := os.Getenv("azure_resource_group")
	nodeImageV, err := az.NodeImageVersion(rg, rg)
	require.NoError(err)
	t.Log("katapolicygen version: ", kataPolicyGenV)
	t.Log("node image version: ", nodeImageV)

	c := kubeclient.NewForTest(t)

	// create the namespace
	ns, err := kuberesource.ResourcesToUnstructured([]any{kuberesource.Namespace(namespace)})
	require.NoError(err)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = c.Apply(ctx, ns...)
	cancel()
	require.NoError(err)
	if namespaceFile != "" {
		require.NoError(os.WriteFile(namespaceFile, []byte(namespace), 0o644))
	}

	// simple deployment that logs the kernel version and then sleeps
	deployment := kuberesource.Deployment(testContainer, "").
		WithSpec(kuberesource.DeploymentSpec().
			WithReplicas(1).
			WithSelector(kuberesource.LabelSelector().WithMatchLabels(
				map[string]string{"app.kubernetes.io/name": testContainer},
			)).
			WithTemplate(kuberesource.PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": testContainer}).
				WithSpec(kuberesource.PodSpec().
					WithContainers(kuberesource.Container().
						WithName(testContainer).
						WithImage("docker.io/bash@sha256:ce062497c248eb1cf4d32927f8c1780cce158d3ed0658c586a5be7308d583cbb").
						WithCommand("/usr/local/bin/bash", "-c", "uname -r; while true; do sleep 10; done"),
					),
				),
			),
		)

	// define resources
	resources := []any{deployment}
	resources = kuberesource.PatchRuntimeHandlers(resources, "kata-cc-isolation")
	resources = kuberesource.PatchNamespaces(resources, namespace)
	resources = kuberesource.PatchImages(resources, imageReplacements)

	toWrite, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)

	// generate policies
	resourceBytes, err := kuberesource.EncodeUnstructured(toWrite)
	require.NoError(err)
	require.NoError(os.WriteFile(path.Join(workdir, "resources.yaml"), resourceBytes, 0o644))
	require.NoError(az.KataPolicyGen(t, path.Join(workdir, "resources.yaml")))

	// load in generated resources and patch the runtime handler again
	resourceBytes, err = os.ReadFile(path.Join(workdir, "resources.yaml"))
	require.NoError(err)
	toApply, err := kubeapi.UnmarshalUnstructuredK8SResource(resourceBytes)
	require.NoError(err)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	err = c.Apply(ctx, toApply...)
	require.NoError(err)
	require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, namespace, testContainer))

	t.Cleanup(func() {
		if skipUndeploy {
			return
		}

		// delete the deployment
		deletePolicy := metav1.DeletePropagationForeground
		require.NoError(c.Client.AppsV1().Deployments(namespace).Delete(context.Background(), testContainer, metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}))
	})

	pods, err := c.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	require.NoError(err)
	pod := pods.Items[0] // only one pod was deployed

	logs, err := c.Client.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).DoRaw(ctx)
	require.NoError(err)
	t.Logf("kernel version in pod %s: %s", pod.Name, string(logs))
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.StringVar(&_platformStr, "platform", "", "Deployment platform")
	flag.BoolVar(&skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}
