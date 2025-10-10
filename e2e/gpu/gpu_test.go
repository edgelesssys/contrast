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
)

const (
	gpuDeploymentName = "gpu-tester"
	gpuName           = "NVIDIA H100 PCIe"
)

// TestGPU runs e2e tests on an GPU-enabled Contrast.
func TestGPU(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.GPU()
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
	t.Run("wait for GPU deployment", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(5*time.Minute))
		defer cancel()

		require := require.New(t)

		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, gpuDeploymentName))

		pods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, gpuDeploymentName)
		require.NoError(err)
		require.Len(pods, 1, "pod not found: %s/%s", ct.Namespace, gpuDeploymentName)
		pod = &pods[0]
	})

	t.Run("check podvm->container mounts by libnvidia-container", func(t *testing.T) {
		require := require.New(t)
		assert := assert.New(t)
		require.NotNil(pod, "require 'wait for GPU deployment' to succeed, no pod found")
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
				stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, pod.Name, argv)
				assert.NoError(err, "running %q:\nstdout:\n%s\nstderr:\n%s", cmd, stdout, stderr)
			}
		}

		const libPath = "/usr/local/nvidia/lib64"
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
			pathThisLib := path.Join(libPath, lib)

			// Run `ls` to check what libraries with that name exist.
			getLibsCmd := fmt.Sprintf("ls %s*", pathThisLib)
			stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, pod.Name, []string{"/usr/bin/env", "bash", "-c", getLibsCmd})
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
				stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, pod.Name, argv)
				assert.NoError(err, "running %q:\nstdout:\n%s\nstderr:\n%s", cmd, stdout, stderr)
			}
		}

		if t.Failed() {
			t.Log("Check log at podvm:/var/log/nvidia-hook.log for errors")
		}
	})

	t.Run("check GPU availability with nvidia-smi", func(t *testing.T) {
		require := require.New(t)
		require.NotNil(pod, "require 'wait for GPU deployment' to succeed, no pod found")
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		argv := []string{"/bin/sh", "-c", "nvidia-smi"}
		stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, pod.Name, argv)
		require.NoError(err, "running nvidia-smi: stdout:\n%s\nstderr:\n%s", stdout, stderr)

		require.Contains(stdout, gpuName, "nvidia-smi output should contain %s", gpuName)
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
