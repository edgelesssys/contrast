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
)

const testContainer = "testcontainer"

func TestAKSRuntime(t *testing.T) {
	require := require.New(t)

	workdir := t.TempDir()

	f, err := os.Open(contrasttest.Flags.ImageReplacementsFile)
	require.NoError(err)
	imageReplacements, err := kuberesource.ImageReplacementsFromFile(f)
	require.NoError(err)
	namespace := contrasttest.MakeNamespace(t, contrasttest.Flags.NamespaceSuffix)

	// Log versions
	kataPolicyGenV, err := az.KataPolicyGenVersion()
	require.NoError(err)
	rg := os.Getenv("azure_resource_group")
	nodeImageV, err := az.NodeImageVersion(context.Background(), rg, rg)
	require.NoError(err)
	t.Log("katapolicygen version: ", kataPolicyGenV)
	t.Log("node image version: ", nodeImageV)

	c := kubeclient.NewForTest(t)

	// create the namespace
	ns, err := kuberesource.ResourcesToUnstructured([]any{kuberesource.Namespace(namespace)})
	require.NoError(err)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	err = c.Apply(ctx, ns...)
	cancel()
	require.NoError(err)
	if contrasttest.Flags.NamespaceFile != "" {
		require.NoError(os.WriteFile(contrasttest.Flags.NamespaceFile, []byte(namespace), 0o644))
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
						WithImage("ghcr.io/edgelesssys/bash@sha256:cabc70d68e38584052cff2c271748a0506b47069ebbd3d26096478524e9b270b").
						WithCommand("/usr/local/bin/bash", "-c", "uname -r; sleep infinity"),
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
	require.NoError(az.KataPolicyGen(path.Join(workdir, "resources.yaml")))

	// load in generated resources
	resourceBytes, err = os.ReadFile(path.Join(workdir, "resources.yaml"))
	require.NoError(err)
	toApply, err := kubeapi.UnmarshalUnstructuredK8SResource(resourceBytes)
	require.NoError(err)

	ctx, cancel = context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	err = c.Apply(ctx, toApply...)
	require.NoError(err)
	require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, namespace, testContainer))

	pods, err := c.PodsFromDeployment(ctx, namespace, testContainer)
	require.NoError(err)
	require.Len(pods, 1)
	pod := pods[0]

	logs, err := c.Client.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).DoRaw(ctx)
	require.NoError(err)
	t.Logf("kernel version in pod %s: %s", pod.Name, string(logs))
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
