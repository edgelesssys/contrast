// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package imagestore

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestImageStore(t *testing.T) {
	tests := map[string]struct {
		name       string
		size       int
		maxOffset  int
		annotation string
		gpu        bool
	}{
		"enabled by default": {
			name:      "imagestore-default",
			size:      10000000,
			maxOffset: 500000,
		},
		"block device size configurable": {
			name:       "imagestore-configured",
			size:       2000000,
			maxOffset:  500000,
			annotation: "2Gi",
		},
		"disabled through annotation": {
			name: "imagestore-disabled",
			// (512MB + 2 * 20MB) / 2: half of the memory is available in the /run tmpfs
			size:       226000,
			maxOffset:  20000,
			annotation: "0",
		},
		"disabled through annotation (GPU)": {
			name: "imagestore-disabled-gpu",
			// (1024MB + 2 * 20MB) / 2
			size:       532000,
			maxOffset:  20000,
			annotation: "0",
			gpu:        true,
		},
	}

	require := require.New(t)

	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(err)

	resources := kuberesource.CoordinatorBundle()

	for _, tc := range tests {
		resources = append(resources, testPod(tc.name, tc.annotation))
	}

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
	defer cancel()
	require.NoError(ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, "coordinator"))

	for name, tc := range tests {
		if platforms.IsGPU(platform) != tc.gpu {
			continue
		}

		t.Run(name, func(_ *testing.T) {
			require.NoError(ct.Kubeclient.WaitForPod(ctx, ct.Namespace, tc.name))

			pod, err := ct.Kubeclient.Client.CoreV1().Pods(ct.Namespace).Get(ctx, tc.name, metav1.GetOptions{})
			require.NoError(err)

			for _, c := range pod.Spec.Containers {
				stdOut, stdErr, err := ct.Kubeclient.ExecContainer(
					ctx,
					ct.Namespace,
					pod.Name,
					c.Name,
					[]string{
						"sh",
						"-c",
						"df /",
					},
				)

				require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
				diskSize, err := extractDiskSize(stdOut)
				require.NoError(err, "failed to extract root disk size from container:\n%s", stdOut)

				require.NoError(checkDiskSize(tc.size, diskSize, tc.maxOffset, c.Name))
			}
		})
	}
}

// extractDiskSize capture the "Size" field from df -h / output.
func extractDiskSize(logs string) (int, error) {
	re := regexp.MustCompile(`(?m)^.*\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)%\s+/$`)
	matches := re.FindStringSubmatch(logs)
	if len(matches) < 2 {
		return 0, fmt.Errorf("disk size not found")
	}
	parsed, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("could not parse disk size: %w", err)
	}
	return parsed, nil
}

// checkDiskSize checks whether the captured size is roughly the same as what we have set.
// The output from df -h will never match exactly what is set in the deployment.
func checkDiskSize(expected, size, maxOffset int, name string) error {
	if size < expected-maxOffset || size > expected+maxOffset {
		return fmt.Errorf("unexpected disk size in container %s: %d (expected about %d)", name, size, expected)
	}
	return nil
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}

func testPod(name, annotation string) any {
	return kuberesource.Pod(name, "").
		WithLabels(map[string]string{"app.kubernetes.io/name": name}).
		WithAnnotations(map[string]string{"contrast.edgeless.systems/image-store-size": annotation}).
		WithSpec(kuberesource.PodSpec().
			WithContainers(
				kuberesource.Container().
					WithName(name+"-1").
					WithImage("ghcr.io/edgelesssys/bash@sha256:cabc70d68e38584052cff2c271748a0506b47069ebbd3d26096478524e9b270b").
					WithCommand("/usr/local/bin/bash", "-c", "sleep infinity").
					WithResources(kuberesource.ResourceRequirements().
						WithMemoryLimitAndRequest(20),
					),
				kuberesource.Container().
					WithName(name+"-2").
					WithImage("ghcr.io/edgelesssys/bash@sha256:cabc70d68e38584052cff2c271748a0506b47069ebbd3d26096478524e9b270b").
					WithCommand("/usr/local/bin/bash", "-c", "sleep infinity").
					WithResources(kuberesource.ResourceRequirements().
						WithMemoryLimitAndRequest(20),
					),
			),
		)
}
