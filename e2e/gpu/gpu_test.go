// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package gpu

import (
	"context"
	"flag"
	"fmt"
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
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

const (
	gpuDeploymentName = "gpu-tester"
	gpuName           = "NVIDIA H100 PCIe"
	nvidiaLibPath     = "/usr/local/nvidia/lib64"
)

// TestGPU runs e2e tests on an GPU-enabled Contrast.
func TestGPU(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	var deviceURI string
	switch platform {
	case platforms.MetalQEMUTDXGPU:
		deviceURI = "nvidia.com/GB100_B200"
	case platforms.MetalQEMUSNPGPU:
		deviceURI = "nvidia.com/GH100_H100_PCIE"
	default:
		t.Errorf("platform %s does not support GPU tests", platform)
	}

	resources := kuberesource.GPU(deviceURI)

	// Since the TDX-GPU testing cluster has multiple GPUs, we run into the drift
	// explained in [1]. To avoid this, we need to remove the deployment of the direct GPU tester,
	// since it wants to claim GPUs based on their CDI IDs, which might not match what's actually
	// available on the node / mounted into the container. This is solved once [2] is released upstream.
	//
	// [1]: https://github.com/edgelesssys/contrast/blob/7cee4f0b2c98be9f5c308adc03f01ab7c5607e85/dev-docs/nvidia/cdi.md?plain=1#L136-L142
	// [2]: https://github.com/kata-containers/kata-containers/pull/12087
	if platform == platforms.MetalQEMUTDXGPU {
		for _, res := range resources {
			if d, ok := res.(*applyappsv1.DeploymentApplyConfiguration); ok {
				containers := d.Spec.Template.Spec.Containers
				d.Spec.Template.Spec.Containers = slices.DeleteFunc(containers,
					func(c applycorev1.ContainerApplyConfiguration) bool {
						return c.Name != nil && *c.Name == "gpu-tester-direct"
					})
			}
		}
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

	var pod *corev1.Pod
	require.True(t, t.Run("wait for GPU deployment", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(5*time.Minute))
		defer cancel()

		require := require.New(t)

		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, gpuDeploymentName))

		pods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, gpuDeploymentName)
		require.NoError(err)
		require.Len(pods, 1, "pod not found: %s/%s", ct.Namespace, gpuDeploymentName)
		pod = &pods[0]
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
		t.Run(fmt.Sprintf("%s: check podvm->container mounts by libnvidia-container", container), func(t *testing.T) {
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

		t.Run(fmt.Sprintf("%s: check GPU availability with nvidia-smi", container), func(t *testing.T) {
			require := require.New(t)
			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
			defer cancel()

			argv := []string{"/bin/sh", "-c", "nvidia-smi"}
			stdout, stderr, err := ct.Kubeclient.ExecContainer(ctx, ct.Namespace, pod.Name, container, argv)
			require.NoError(err, "running nvidia-smi: stdout:\n%s\nstderr:\n%s", stdout, stderr)

			require.Contains(stdout, gpuName, "nvidia-smi output should contain %s", gpuName)
		})
	}

	for _, container := range nonGPUContainers {
		t.Run(fmt.Sprintf("%s: check that path %s is not available", nvidiaLibPath, container), func(t *testing.T) {
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
