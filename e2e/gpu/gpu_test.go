// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// runs-on: Metal-QEMU-SNP-GPU,Metal-QEMU-TDX-GPU

//go:build e2e

package gpu

import (
	"context"
	"flag"
	"fmt"
	"maps"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	gpuDeploymentName = "gpu-tester"
	nvidiaLibPath     = "/usr/local/nvidia/lib64"
)

var expectedGPUModels = map[corev1.ResourceName]string{
	"nvidia.com/GB100_B200":         "B200",
	"nvidia.com/GB110_B300_SXM6_AC": "B300",
	"nvidia.com/GH100_H100_PCIE":    "H100",
	// TODO(msanft): remove when the GPU operator is updated in all clusters.
	"nvidia.com/pgpu": "H100",
}

type gpuConfig struct {
	resource corev1.ResourceName
	model    string
	quantity int64
}

func (c gpuConfig) deploymentName() string {
	name := gpuDeploymentName + "-" + strings.ToLower(c.model)
	if c.quantity > 1 {
		name += "-multi-gpu"
	}
	return name
}

// TestGPU runs e2e tests on an GPU-enabled Contrast.
func TestGPU(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	require.True(t, platforms.IsGPU(platform), "platform %s does not support GPU tests", platform)

	nodes, err := ct.Kubeclient.Client.CoreV1().Nodes().List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	gpuConfigs, err := gpuConfigsFromNodes(nodes.Items)
	require.NoError(t, err)

	var resources []any
	for i, config := range gpuConfigs {
		t.Logf("Using GPU resource %s (%s) with quantity %d", config.resource, config.model, config.quantity)
		// The hostpath CSI volume is node-local. Exercise the block-device
		// regression once without preventing other GPU models from running on
		// their respective nodes.
		resources = append(resources, kuberesource.GPU(config.deploymentName(), string(config.resource), config.quantity, i == 0)...)
	}

	coordinator := kuberesource.CoordinatorBundle()

	resources = append(resources, coordinator...)

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	for _, config := range gpuConfigs {
		t.Run(config.model, func(t *testing.T) {
			testGPUDeployment(t, ct, config)
		})
	}
}

func testGPUDeployment(t *testing.T, ct *contrasttest.ContrastTest, config gpuConfig) {
	deploymentName := config.deploymentName()
	expectedGPUModel := config.model
	expectedGPUAmount := config.quantity

	var pod corev1.Pod
	require.True(t, t.Run("wait for GPU deployment", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(5*time.Minute))
		defer cancel()

		require := require.New(t)

		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, deploymentName))

		deploymentPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, deploymentName)
		require.NoError(err)
		require.Len(deploymentPods, 1, "expected 1 pod for %s/%s", ct.Namespace, deploymentName)
		pod = deploymentPods[0]
	}), "GPU deployment needs to succeed for subsequent tests")

	var gpuContainers, nonGPUContainers []string
	for _, container := range pod.Spec.Containers {
		if shouldHaveGPU(container) {
			gpuContainers = append(gpuContainers, container.Name)
		} else {
			nonGPUContainers = append(nonGPUContainers, container.Name)
		}
	}

	for _, container := range gpuContainers {
		t.Run(fmt.Sprintf("%s/%s: check podvm->container mounts by libnvidia-container", pod.Name, container), func(t *testing.T) {
			assert := assert.New(t)
			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(3*time.Minute))
			defer cancel()

			expectBins := []string{
				// Binaries taken from output of libnvidia-container v1.17.8.
				"nvidia-smi",
				"nvidia-debugdump",
				"nvidia-cuda-mps-control",
				"nvidia-cuda-mps-server",
			}
			for _, bin := range expectBins {
				for _, cmd := range []string{
					fmt.Sprintf("[[ $(command -v %s) == /usr/bin/%s ]]", bin, bin),
					fmt.Sprintf("test -x /usr/bin/%s", bin),
				} {
					argv := []string{"/usr/bin/env", "bash", "-c", cmd}
					stdout, stderr, err := ct.Kubeclient.ExecContainer(ctx, ct.Namespace, pod.Name, container, argv)
					assert.NoError(err, "running %q:\nstdout:\n%s\nstderr:\n%s", cmd, stdout, stderr)
				}
			}

			expectLibs := map[string]struct {
				abiLink   bool // Wether to expect a link with ABI version, like .so.1
				unverLink bool // Wether to expect a link without any version, like .so
			}{
				// Libraries taken from output of libnvidia-container v1.17.8.
				"libnvidia-ml.so":              {abiLink: true},
				"libnvidia-cfg.so":             {abiLink: true},
				"libcuda.so":                   {abiLink: true, unverLink: true},
				"libcudadebugger.so":           {abiLink: true},
				"libnvidia-opencl.so":          {abiLink: true},
				"libnvidia-gpucomp.so":         {abiLink: false},
				"libnvidia-ptxjitcompiler.so":  {abiLink: true},
				"libnvidia-allocator.so":       {abiLink: true},
				"libnvidia-pkcs11-openssl3.so": {abiLink: false},
				"libnvidia-nvvm.so":            {abiLink: true},
			}
			for lib, libChecks := range expectLibs {
				pathThisLib := path.Join(nvidiaLibPath, lib)

				// Run `ls` to check what libraries with that name exist.
				getLibsCmd := fmt.Sprintf("ls %s*", pathThisLib)
				stdout, stderr, err := ct.Kubeclient.ExecContainer(ctx, ct.Namespace, pod.Name, container, []string{"/usr/bin/env", "bash", "-c", getLibsCmd})
				assert.NoError(err, "running %q:\nstdout:\n%s\nstderr:\n%s", getLibsCmd, stdout, stderr)
				stdout = strings.TrimSpace(stdout)
				var lsLibs []string
				if stdout != "" {
					lsLibs = strings.Split(stdout, "\n")
				}
				if !assert.NotEmpty(lsLibs, "expected at least one library for %q", lib) {
					continue
				}

				// Determine what library paths we got.
				fullVerRegex := regexp.MustCompile(fmt.Sprintf(`%s\.\d+(\.\d+)+$`, strings.ReplaceAll(pathThisLib, ".", "\\.")))
				abiVerRegex := regexp.MustCompile(fmt.Sprintf(`%s\.\d+$`, strings.ReplaceAll(pathThisLib, ".", "\\.")))
				unverRegex := regexp.MustCompile(fmt.Sprintf(`%s$`, strings.ReplaceAll(pathThisLib, ".", "\\.")))
				fullVerPath := ""
				abiVerPath := ""
				unverPath := ""
				for _, libPath := range lsLibs {
					switch {
					case fullVerRegex.MatchString(libPath):
						fullVerPath = libPath
					case abiVerRegex.MatchString(libPath):
						abiVerPath = libPath
					case unverRegex.MatchString(libPath):
						unverPath = libPath
					default:
					}
				}

				// Ensure library can be executed.
				cmds := []string{fmt.Sprintf("test -x %s", fullVerPath)}
				if libChecks.abiLink && assert.NotEmpty(abiVerPath, "expected ABI versioned link for %q in %v", lib, lsLibs) {
					// Ensure correct link from .so.1 to .so.570.169
					cmds = append(cmds, fmt.Sprintf("[[ $(realpath %s) == %s ]] ", abiVerPath, fullVerPath))
				}
				if libChecks.unverLink && assert.NotEmpty(unverPath, "expected unversioned link for %q in %v", lib, lsLibs) {
					// Ensure correct link from .so to .so.570.169
					cmds = append(cmds, fmt.Sprintf("[[ $(realpath %s) == %s ]]", unverPath, fullVerPath))
				}

				for _, cmd := range cmds {
					argv := []string{"/usr/bin/env", "bash", "-c", cmd}
					stdout, stderr, err := ct.Kubeclient.ExecContainer(ctx, ct.Namespace, pod.Name, container, argv)
					assert.NoError(err, "running %q:\nstdout:\n%s\nstderr:\n%s", cmd, stdout, stderr)
				}
			}
		})

		t.Run(fmt.Sprintf("%s/%s: check GPU availability with nvidia-smi", pod.Name, container), func(t *testing.T) {
			require := require.New(t)
			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
			defer cancel()

			argv := []string{"/bin/sh", "-c", "nvidia-smi -L"}
			stdout, stderr, err := ct.Kubeclient.ExecContainer(ctx, ct.Namespace, pod.Name, container, argv)
			require.NoError(err, "running nvidia-smi -L: stdout:\n%s\nstderr:\n%s", stdout, stderr)
			gpuLines := strings.Split(strings.TrimSpace(stdout), "\n")

			require.Len(gpuLines, int(expectedGPUAmount), "expected %d GPUs, got %d:\n%s", expectedGPUAmount, len(gpuLines), stdout)
			for i, line := range gpuLines {
				require.Regexp(`^GPU [0-9]+: NVIDIA .+ \(UUID: GPU-[^)]+\)$`, line, "GPU %d has unexpected nvidia-smi output", i)
				require.Contains(line, expectedGPUModel, "GPU %d should be an NVIDIA %s", i, expectedGPUModel)
			}
		})
	}

	for _, container := range nonGPUContainers {
		t.Run(fmt.Sprintf("%s/%s: check that path %s is not available", pod.Name, container, nvidiaLibPath), func(t *testing.T) {
			require := require.New(t)
			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
			defer cancel()

			argv := []string{"/bin/test", "!", "-d", nvidiaLibPath}
			stdout, stderr, err := ct.Kubeclient.ExecContainer(ctx, ct.Namespace, pod.Name, container, argv)
			require.NoError(err, "path %q should not exist, but does:\nstdout:\n%s\nstderr:\n%s", nvidiaLibPath, stdout, stderr)
		})
	}
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}

// gpuConfigsFromNodes determines the GPU resources and models advertised by the
// sandbox device plugin and whether the test can exercise multi-GPU
// passthrough with 2 GPUs. Multiple resources for the same model, such as the
// H100-specific resource and pgpu alias, are only tested once.
func gpuConfigsFromNodes(nodes []corev1.Node) ([]gpuConfig, error) {
	gpuClasses := slices.Collect(maps.Keys(expectedGPUModels))
	slices.Sort(gpuClasses)
	var configs []gpuConfig
	testedModels := map[string]struct{}{}
	for _, gpuClass := range gpuClasses {
		model := expectedGPUModels[gpuClass]
		if _, ok := testedModels[model]; ok {
			continue
		}

		var maxQuantity int64
		for _, node := range nodes {
			quantity, ok := node.Status.Allocatable[gpuClass]
			if ok && quantity.Value() > maxQuantity {
				maxQuantity = quantity.Value()
			}
		}
		if maxQuantity > 0 {
			configs = append(configs, gpuConfig{
				resource: gpuClass,
				model:    model,
				quantity: min(maxQuantity, int64(2)),
			})
			testedModels[model] = struct{}{}
		}
	}
	if len(configs) > 0 {
		return configs, nil
	}

	// No supported class found, so build useful diagnostics.
	var advertised []string
	for _, node := range nodes {
		for resourceName, quantity := range node.Status.Allocatable {
			if strings.HasPrefix(resourceName.String(), "nvidia.com/") && quantity.Value() > 0 {
				advertised = append(advertised, resourceName.String())
			}
		}
	}
	slices.Sort(advertised)
	advertised = slices.Compact(advertised)
	return nil, fmt.Errorf("no supported GPU resource found; advertised NVIDIA resources: %v", advertised)
}

// shouldHaveGPU decides whether a container should have received a GPU mount.
// This could be true either because it explicitly requested a GPU resource, or because it sets the
// magic environment variable.
func shouldHaveGPU(container corev1.Container) bool {
	if slices.ContainsFunc(container.Env, func(envVar corev1.EnvVar) bool {
		return envVar.Name == "NVIDIA_VISIBLE_DEVICES" && envVar.Value == "all"
	}) {
		return true
	}

	for resource := range container.Resources.Limits {
		if strings.HasPrefix(resource.String(), "nvidia.com/") {
			return true
		}
	}

	return false
}
