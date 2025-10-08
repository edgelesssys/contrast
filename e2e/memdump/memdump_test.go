// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package memdump

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

const (
	listenerDeployment = "listener"
	senderDeployment   = "sender"
	memdumpDeployment  = "memdump"

	canaryString = "deadbeafcafebabe0123456789abcdef"
	nsenterCmd   = `
	nsenter --target 1 --mount -- %s -n k8s.io c ls -q 'labels."io.kubernetes.pod.namespace"==%q,labels."io.kubernetes.pod.name"==%q,labels."io.cri-containerd.kind"=="sandbox"' |
		xargs -I {} pgrep -f sandbox-{} |
		xargs -I {} gcore -o /memdump {} >/dev/null &&
		strings /memdump.*`
)

func TestMemDump(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.MemDump()
	coordinator := kuberesource.CoordinatorBundle()

	resources = append(resources, coordinator...)

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources, platform)

	memdumpTester := kuberesource.MemDumpTester()
	memdumpTester = kuberesource.PatchImages(memdumpTester, ct.ImageReplacements)
	memdumpTester = kuberesource.PatchNamespaces(memdumpTester, ct.Namespace)
	memdumpUnstructured, err := kuberesource.ResourcesToUnstructured(memdumpTester)
	require.NoError(t, err)
	require.NoError(t, ct.Kubeclient.Apply(t.Context(), memdumpUnstructured...))

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	t.Run("memory dump does not contain canary string", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		require := require.New(t)

		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, listenerDeployment))
		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, senderDeployment))

		senderPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, senderDeployment)
		require.NoError(err)
		require.Len(senderPods, 1, "pod not found: %s/%s", ct.Namespace, senderDeployment)

		// Send canary string from sender to listener via socat over TCP, encrypted via Contrast Service Mesh
		argv := []string{"/bin/sh", "-c", "printf %s '" + canaryString + "' | socat - TCP:127.137.0.1:8000"}
		_, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, senderPods[0].Name, argv)
		require.NoError(err, "stderr: %q", stderr)

		listenerPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, listenerDeployment)
		require.NoError(err)
		require.Len(listenerPods, 1, "pod not found: %s/%s", ct.Namespace, listenerDeployment)

		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, memdumpDeployment))
		memdumpPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, memdumpDeployment)
		require.NoError(err)
		require.Len(memdumpPods, 1, "pod not found: %s/%s", ct.Namespace, memdumpDeployment)

		ctrCmd := "k3s ctr"
		_, stderr, err = ct.Kubeclient.Exec(ctx, ct.Namespace, memdumpPods[0].Name, []string{"/bin/sh", "-c", "nsenter -t 1 -m -- which k3s"})
		if err != nil && len(stderr) == 0 {
			ctrCmd = "ctr"
		} else if err != nil {
			require.NoError(err, "stderr: %q", stderr)
		}

		// Create core dump of the qemu process of the listener pod and search for the canary string in the core dump.
		// The canary string must not be present in the core dump.
		argv = []string{
			"/bin/sh",
			"-c",
			fmt.Sprintf(nsenterCmd, ctrCmd, ct.Namespace, listenerPods[0].Name),
		}
		stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, memdumpPods[0].Name, argv)
		require.NoError(err, "stderr: %q", stderr)

		require.NotContains(stdout, canaryString, "canary string found in memory dump")

		// Verify that the listener received the canary string
		argv = []string{"/bin/sh", "-c", "cat /dev/shm/data"}
		stdout, stderr, err = ct.Kubeclient.Exec(ctx, ct.Namespace, listenerPods[0].Name, argv)
		require.NoError(err, "stderr: %q", stderr)
		require.Equal(canaryString, stdout, "canary string not received by listener")
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
