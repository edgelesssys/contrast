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

	req "github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestImageStore(t *testing.T) {
	tests := map[string]struct {
		name       string
		annotation string
	}{
		"enabled by default": {
			name: "imagestore-default",
		},
		"block device size configurable": {
			name:       "imagestore-configured",
			annotation: "2Gi",
		},
		"disabled through annotation": {
			name:       "imagestore-disabled",
			annotation: "0",
		},
	}

	require := req.New(t)

	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(err)

	resources := kuberesource.CoordinatorBundle()

	for _, tc := range tests {
		resources = append(resources, testPod(tc.name, tc.annotation))
	}

	resources = append(resources, postgresPod())

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)
	resources = kuberesource.AddImageStore(resources)

	ct.Init(t, resources)

	require.True(t.Run("generate", func(t *testing.T) {
		require.NoError(ct.RunGenerate(t.Context(), "--inject-image-store", "--calculate-pod-memory"))
	}), "contrast generate needs to succeed for subsequent tests")
	require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
			t.Cleanup(cancel)

			require = req.New(t)
			require.NoError(ct.Kubeclient.WaitForPod(ctx, ct.Namespace, tc.name))

			pod, err := ct.Kubeclient.Client.CoreV1().Pods(ct.Namespace).Get(ctx, tc.name, metav1.GetOptions{})
			require.NoError(err)

			var expectedKibiBytes int
			switch tc.annotation {
			case "0":
				require.NotNil(pod.Spec.Resources)
				require.NotNil(pod.Spec.Resources.Limits)
				require.NotNil(pod.Spec.Resources.Limits.Memory())
				podMemory := int(pod.Spec.Resources.Limits.Memory().Value())

				containerMemory := 0
				for _, c := range pod.Spec.Containers {
					memory := c.Resources.Limits[corev1.ResourceMemory]
					containerMemory += int(memory.Value())
				}
				for _, c := range pod.Spec.InitContainers {
					if c.RestartPolicy == nil || *c.RestartPolicy != corev1.ContainerRestartPolicyAlways {
						continue
					}
					memory := c.Resources.Limits[corev1.ResourceMemory]
					containerMemory += int(memory.Value())
				}

				initContainerMemory := 0
				for _, c := range pod.Spec.InitContainers {
					if c.RestartPolicy != nil && *c.RestartPolicy == corev1.ContainerRestartPolicyAlways {
						continue
					}
					memory := c.Resources.Limits[corev1.ResourceMemory]
					initContainerMemory = max(initContainerMemory, int(memory.Value()))
				}

				// Usually, K8s adjusts the available memory to either be the maximum of (sequential) init containers,
				// or the sum of (concurrent) normal and sidecar containers
				minimumMemory := max(initContainerMemory, containerMemory)
				// Only 50% of the allocated memory is available in /run. Half of the calculated pod memory should then
				// accommodate for the container limits as defined above, plus additional memory to pull the images.
				require.Greater(podMemory/2, minimumMemory, "pod memory should be greater than the container limits")

				defaultMemory := platforms.DefaultMemoryInMebiBytes(platform) * 1024 * 1024
				totalMemory := defaultMemory + podMemory
				// When the imagestore is disabled, the expectedKibiBytes are (the total VM memory in KiB - internal VM overhead) / 2
				// For more information on the internal VM overhead, see https://github.com/edgelesssys/contrast/pull/1196
				vmOverheadBytes := 99 * 1024 * 1024
				vmOverheadBytes += swiotlbSizeBytes(totalMemory)
				expectedKibiBytes = ((totalMemory - vmOverheadBytes) / 1024) / 2
			case "":
				size := resource.MustParse("10Gi")
				expectedKibiBytes = int(size.Value()) / 1024
				require.Nil(pod.Spec.Resources)
			default:
				size := resource.MustParse(tc.annotation)
				expectedKibiBytes = int(size.Value()) / 1024
				require.Nil(pod.Spec.Resources)
			}

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

				require.NoError(checkDiskSize(expectedKibiBytes, diskSize, c.Name))
			}
		})
	}

	t.Run("large image with pod resource limits", func(t *testing.T) {
		// Increase timeout, as pulling the image may take a while.
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(5*time.Minute))
		t.Cleanup(cancel)

		require = req.New(t)

		// The uncompressed and compressed postgres image layers together are ~815Mi.
		// The largest VM memory when not using auto-memory limits can be calculated as:
		//   - default memory (GPU): 1024Mi
		//   - debugshell memory limit: 1000Mi
		//   - container memory limit: 10Mi
		// The available memory for image pulling is then (1024+1000+10) / 2 = 1017Mi.
		// The debugshell image is ~500-600Mi, which leaves ~400-500Mi for pulling the postgres image,
		// so the pod wouldn't be able to start.
		// This test verifies that the added pod memory limit during generate is enough to pull the image
		// when the imagestore is disbabled. If the pod comes up, the test passes.
		require.NoError(ct.Kubeclient.WaitForPod(ctx, ct.Namespace, "postgres"))
	})
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

// swiotlbSizeBytes returns the SWIOTLB bounce buffer reservation the guest kernel makes on confidential platforms, given the total VM memory in bytes.
// The kernel sizes them at 6% of total RAM, and clamps the result to [64MiB, SZ_1G], then SWIOTLB rounds it up to a power of two.
// https://github.com/torvalds/linux/blob/adc218676eef25575469234709c2d87185ca223a/arch/x86/mm/mem_encrypt.c#L115-L134
func swiotlbSizeBytes(totalMemoryBytes int) int {
	const (
		ioTLBDefaultSize = 64 * 1024 * 1024   // IO_TLB_DEFAULT_SIZE
		maxSize          = 1024 * 1024 * 1024 // SZ_1G
	)
	size := min(max(totalMemoryBytes*6/100, ioTLBDefaultSize), maxSize)
	pow2 := ioTLBDefaultSize
	for pow2 < size {
		pow2 *= 2
	}
	return pow2
}

// checkDiskSize checks whether the captured size is roughly the same as what we have set.
// The output from df -h will never match exactly what is set in the deployment.
func checkDiskSize(expected, size int, name string) error {
	maxOffset := int(0.1 * float32(expected))
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
		WithAnnotations(map[string]string{kuberesource.ImageStoreSizeAnnotationKey: annotation}).
		WithSpec(
			kuberesource.PodSpec().
				WithContainers(
					kuberesource.Container().
						WithName(name+"-1").
						WithImage("ghcr.io/edgelesssys/bash@sha256:cabc70d68e38584052cff2c271748a0506b47069ebbd3d26096478524e9b270b").
						WithCommand("/usr/local/bin/bash", "-c", "sleep infinity").
						WithResources(
							kuberesource.ResourceRequirements().
								WithMemoryLimitAndRequest(40),
						),
					kuberesource.Container().
						WithName(name+"-2").
						WithImage("ghcr.io/edgelesssys/bash@sha256:cabc70d68e38584052cff2c271748a0506b47069ebbd3d26096478524e9b270b").
						WithCommand("/usr/local/bin/bash", "-c", "sleep infinity").
						WithResources(
							kuberesource.ResourceRequirements().
								WithMemoryLimitAndRequest(40),
						),
				),
		)
}

func postgresPod() any {
	return kuberesource.Pod("postgres", "").
		WithAnnotations(map[string]string{kuberesource.ImageStoreSizeAnnotationKey: "0"}).
		WithSpec(
			kuberesource.PodSpec().
				WithContainers(
					kuberesource.Container().
						WithName("postgres").
						WithImage("docker.io/library/postgres@sha256:4aabea78cf39b90e834caf3af7d602a18565f6fe2508705c8d01aa63245c2e20").
						WithCommand("/bin/bash", "-c", "sleep infinity").
						WithResources(
							kuberesource.ResourceRequirements().
								WithMemoryLimitAndRequest(10),
						).
						WithVolumeMounts(
							kuberesource.VolumeMount().
								WithName("postgres-data").
								WithMountPath("/var/lib/postgresql"),
						),
				).
				WithVolumes(
					kuberesource.Volume().
						WithName("postgres-data").
						WithEmptyDir(kuberesource.EmptyDirVolumeSource().Inner()),
				),
		)
}
