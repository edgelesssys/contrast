// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package imagepullerauth

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
	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

func TestImagepullerAuth(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)
	imagePullerConfig, err := createImagepullerConfig()
	require.NoError(t, err)
	ct.NodeInstallerImagePullerConfig = imagePullerConfig

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.CoordinatorBundle()
	deploymentName := "auth-test"
	deployment := kuberesource.Deployment(deploymentName, ct.Namespace).
		WithSpec(kuberesource.DeploymentSpec().
			WithReplicas(1).
			WithSelector(kuberesource.LabelSelector().
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": deploymentName}),
			).
			WithTemplate(kuberesource.PodTemplateSpec().
				WithLabels(map[string]string{"app.kubernetes.io/name": deploymentName}).
				WithSpec(kuberesource.PodSpec().
					WithContainers(
						kuberesource.Container().
							WithName("my-image-is-private").
							WithImage("ghcr.io/edgelesssys/bash-private@sha256:44ddf003cf6d966487da334edf972c55e91d1aa30db5690ad0445b459cbca924").
							WithCommand("bash", "-c", "sleep infinity").
							WithResources(kuberesource.ResourceRequirements().
								WithMemoryLimitAndRequest(100),
							),
					),
				),
			),
		)

	resources = append(resources, deployment)
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.NoError(t, err)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	require.NoError(t, ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, deploymentName))
}

func createImagepullerConfig() ([]byte, error) {
	token := os.Getenv("CONTRAST_GHCR_READ")

	cfg := map[string]any{
		"registries": map[string]any{
			"ghcr.io.": map[string]string{
				"auth": token,
			},
		},
	}

	return toml.Marshal(cfg)
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
