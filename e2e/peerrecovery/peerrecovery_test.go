// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

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
	"testing"
	"time"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// TestWorkloadSecrets tests that secrets are correctly injected into workloads.
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
		ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(3*time.Minute))
		t.Cleanup(cancel)

		require.NoError(ct.Kubeclient.ScaleStatefulSet(ctx, ct.Namespace, "coordinator", int32(numCoordinators)))
		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.StatefulSet{}, ct.Namespace, "coordinator"))
	}), "coordinator needs to scale up for subsequent tests to run")

	t.Run("coordinators recover", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(2*time.Minute))
		t.Cleanup(cancel)

		workspaceRoot := t.TempDir()
		manifestBytes, err := os.ReadFile(path.Join(ct.WorkDir, "manifest.json"))
		require.NoError(err)

		f, err := os.Open(ct.ImageReplacementsFile)
		require.NoError(err)
		imageReplacements, err := kuberesource.ImageReplacementsFromFile(f)
		require.NoError(err)

		pods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "coordinator")
		require.NoError(err)
		require.Len(pods, 3)

		for _, pod := range pods {
			forwarder := kuberesource.PortForwarder(pod.Name, ct.Namespace).
				WithListenPorts([]int32{1313}).
				WithForwardTarget(pod.Status.PodIP).
				PodApplyConfiguration
			forwarder = kuberesource.PatchImages([]any{forwarder}, imageReplacements)[0].(*v1.PodApplyConfiguration)

			_, err := ct.Kubeclient.Client.CoreV1().Pods(ct.Namespace).Apply(ctx, forwarder, metav1.ApplyOptions{FieldManager: "peerrecovery_test"})
			require.NoError(err)
			workspace := path.Join(workspaceRoot, pod.Name)
			require.NoError(os.Mkdir(workspace, 0o777))
			require.NoError(os.WriteFile(path.Join(workspace, "manifest.json"), manifestBytes, 0o666))
		}

		require.EventuallyWithT(func(collect *assert.CollectT) {
			assert := assert.New(collect)
			for _, pod := range pods {
				cmd := cmd.NewVerifyCmd()
				cmd.Flags().String("workspace-dir", path.Join(workspaceRoot, pod.Name), "")
				cmd.Flags().String("log-level", "debug", "")
				assert.NoErrorf(ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-"+pod.Name, "1313", func(addr string) error {
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
