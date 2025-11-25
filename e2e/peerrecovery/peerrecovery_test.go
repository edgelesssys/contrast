// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package peerrecovery

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// TestPeerRecovery tests that Coordinators started after the first Coordinator
// recover automatically from the existing peer.
func TestPeerRecovery(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.CoordinatorBundle()

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	numCoordinators := 3
	require.True(t, t.Run("scale coordinator", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(3*time.Minute))
		t.Cleanup(cancel)

		require.NoError(ct.Kubeclient.ScaleStatefulSet(ctx, ct.Namespace, "coordinator", int32(numCoordinators)))
		require.NoError(ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, "coordinator"))
	}), "coordinator needs to scale up for subsequent tests to run")

	t.Run("coordinators recover", func(t *testing.T) {
		// Now that the Coordinator is scaled up, we verify that each of the additional instances
		// recovers eventually, by running verify against each of the individual pods, from a
		// fresh directory to rule out any interference between the verify calls.
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		t.Cleanup(cancel)

		// Read the generated manifest for subsequent verify steps in per-pod subdirectories.
		manifestBytes, err := os.ReadFile(path.Join(ct.WorkDir, "manifest.json"))
		require.NoError(err)

		// Create a temporary directory for output of verify commands.
		workspaceRoot := t.TempDir()

		// Fetch image replacements for patching the port-forwarder pods.
		f, err := os.Open(ct.ImageReplacementsFile)
		require.NoError(err)
		imageReplacements, err := kuberesource.ImageReplacementsFromFile(f)
		require.NoError(err)

		// Get the list of pods to verify.
		pods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "coordinator")
		require.NoError(err)
		require.Len(pods, 3)

		userapiPort, err := strconv.Atoi(userapi.Port)
		require.NoError(err)

		for _, pod := range pods {
			// Apply a port-forwarder that targets only the current iteration's pod under test.
			forwarder := kuberesource.PortForwarder(pod.Name, ct.Namespace).
				WithListenPorts([]int32{int32(userapiPort)}).
				WithForwardTarget(pod.Status.PodIP).
				PodApplyConfiguration
			forwarder, ok := kuberesource.PatchImages([]any{forwarder}, imageReplacements)[0].(*applycorev1.PodApplyConfiguration)
			require.True(ok)

			_, err := ct.Kubeclient.Client.CoreV1().Pods(ct.Namespace).Apply(ctx, forwarder, metav1.ApplyOptions{FieldManager: "peerrecovery_test"})
			require.NoError(err)

			// Copy the manifest to the subdirectory of this pod.
			workspace := path.Join(workspaceRoot, pod.Name)
			require.NoError(os.Mkdir(workspace, 0o777))
			require.NoError(os.WriteFile(path.Join(workspace, "manifest.json"), manifestBytes, 0o600))
		}

		require.EventuallyWithT(func(collect *assert.CollectT) {
			assert := assert.New(collect)
			for _, pod := range pods {
				// Run verify from the pod's workspace subdir against the pod's dedicated forwarder.
				cmd := cmd.NewVerifyCmd()
				cmd.Flags().String("workspace-dir", path.Join(workspaceRoot, pod.Name), "")
				cmd.Flags().String("log-level", "debug", "")
				assert.NoErrorf(ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-"+pod.Name, userapi.Port, func(addr string) error {
					args := []string{
						"--coordinator", addr,
					}
					cmd.SetArgs(args)
					cmd.SetOut(io.Discard)
					errBuf := &bytes.Buffer{}
					cmd.SetErr(errBuf)

					if err := cmd.Execute(); err != nil {
						return fmt.Errorf("running %q: %s", cmd.Use, errBuf)
					}
					return nil
				}), "pod %s not recovered", pod.Name)
			}
		}, 2*time.Minute, 5*time.Second)
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
