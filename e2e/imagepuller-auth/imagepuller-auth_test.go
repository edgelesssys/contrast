// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package imagepullerauth

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestImagepullerAuth(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

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

	t.Run("private image can be pulled", func(t *testing.T) {
		ctx := t.Context()
		require.NoError(t, ct.Kubeclient.WaitForCoordinator(ctx, ct.Namespace))
		require.NoError(t, createImagepullerConfig(ctx, ct))
		require.NoError(t, restartDaemonSet(ctx, ct, fmt.Sprintf("%s-nodeinstaller", ct.RuntimeClassName)))
		require.NoError(t, ct.Kubeclient.WaitForDaemonSet(ctx, ct.Namespace, fmt.Sprintf("%s-nodeinstaller", ct.RuntimeClassName)))
		require.NoError(t, ct.Kubeclient.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, deploymentName))
		require.NoError(t, ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, deploymentName))
	})
}

func createImagepullerConfig(ctx context.Context, ct *contrasttest.ContrastTest) error {
	token := os.Getenv("CONTRAST_GHCR_READ")

	cfg := map[string]any{
		"registries": map[string]any{
			"ghcr.io.": map[string]string{
				"auth": token,
			},
		},
	}

	tomlData, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal toml: %w", err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contrast-node-installer-imagepuller-config",
			Namespace: ct.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"contrast-imagepuller.toml": tomlData,
		},
	}

	_, err = ct.Kubeclient.Client.CoreV1().Secrets(ct.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	return err
}

// restartDaemonSet forces a rollout restart of the node-installer DaemonSet.
func restartDaemonSet(ctx context.Context, ct *contrasttest.ContrastTest, name string) error {
	client := ct.Kubeclient.Client.AppsV1().DaemonSets(ct.Namespace)

	ds, err := client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting daemonset: %w", err)
	}

	// Trigger a restart by updating an annotation.
	if ds.Spec.Template.Annotations == nil {
		ds.Spec.Template.Annotations = map[string]string{}
	}
	ds.Spec.Template.Annotations["contrast/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = client.Update(ctx, ds, metav1.UpdateOptions{})
	return err
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
