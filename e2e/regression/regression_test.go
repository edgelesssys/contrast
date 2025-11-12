// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

// regression runs a series of regression tests that ensure we don't reintroduce bugs we fixed in the past.
//
// The test cycles through the directories in testdata and runs the entire Contrast workload
// lifecycle for each directory.
package regression

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applybatchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

func TestRegression(t *testing.T) {
	dataDir := "./e2e/regression/testdata/"
	dirs, err := os.ReadDir(dataDir)
	require.NoError(t, err)

	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	ct := contrasttest.New(t)

	// Initially just deploy the coordinator bundle

	resources := kuberesource.CoordinatorBundle()
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		t.Run(dir.Name(), func(t *testing.T) {
			require := require.New(t)

			c := kubeclient.NewForTest(t)

			pattern := filepath.Join(dataDir, dir.Name(), "*.yml")
			files, err := filepath.Glob(pattern)
			require.NoError(err)

			containsCronJob := false
			intermediateResources := resources
			for _, file := range files {
				resourceYAML, err := os.ReadFile(file)
				require.NoError(err)
				resourceYAML = bytes.ReplaceAll(resourceYAML, []byte("@@REPLACE_NAMESPACE@@"), []byte(ct.Namespace))

				newResources, err := kuberesource.UnmarshalApplyConfigurations(resourceYAML)
				require.NoError(err)

				newResources = kuberesource.PatchRuntimeHandlers(newResources, runtimeHandler)

				// Check if we are testing a cron job
				unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
				require.NoError(err)
				for _, resource := range unstructuredResources {
					if resource.GetKind() == "CronJob" {
						containsCronJob = true
					}
				}

				// write the new resources
				base := path.Base(file)
				resourceBytes, err := kuberesource.EncodeResources(newResources...)
				require.NoError(err)
				require.NoError(os.WriteFile(path.Join(ct.WorkDir, base), resourceBytes, 0o644))
				t.Cleanup(func() {
					require.NoError(os.Remove(path.Join(ct.WorkDir, base)))
				})

				intermediateResources = append(intermediateResources, newResources...)
			}

			// generate, set, deploy and verify the new policy
			require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

			if containsCronJob {
				return
			}

			require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
			require.True(t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
			require.True(t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")
			require.True(t.Run("wait-for-resource", func(t *testing.T) {
				assert := assert.New(t)
				for _, resource := range intermediateResources {
					t.Cleanup(func() {
						if err := cleanupResource(t.Context(), resource, ct); err != nil {
							t.Logf("failed to delete resource: %s:", err)
						}
					})
					ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(3*time.Minute))
					defer cancel()
					switch r := resource.(type) {
					case *applyappsv1.DeploymentApplyConfiguration:
						assert.NoError(c.WaitForDeployment(ctx, ct.Namespace, *r.Name))
					case *applyappsv1.DaemonSetApplyConfiguration:
						assert.NoError(c.WaitForDaemonSet(ctx, ct.Namespace, *r.Name))
					case *applycorev1.PodApplyConfiguration:
						assert.NoError(c.WaitForPod(ctx, ct.Namespace, *r.Name))
					case *applybatchv1.JobApplyConfiguration:
						assert.NoError(c.WaitForJob(ctx, ct.Namespace, *r.Name))
					case *applyappsv1.ReplicaSetApplyConfiguration:
						assert.NoError(c.WaitForReplicaSet(ctx, ct.Namespace, *r.Name))
					case *applycorev1.ReplicationControllerApplyConfiguration:
						assert.NoError(c.WaitForReplicationController(ctx, ct.Namespace, *r.Name))
					}
				}
			}))
		})
	}
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()
	os.Exit(m.Run())
}

func cleanupResource(ctx context.Context, resource any, ct *contrasttest.ContrastTest) error {
	unstructuredResources, err := kuberesource.ResourcesToUnstructured([]any{resource})
	if err != nil {
		return err
	}
	bgDeletion := metav1.DeletePropagationForeground
	for _, r := range unstructuredResources {
		if strings.Contains(r.GetName(), "coordinator") {
			return nil
		}
		client, err := ct.Kubeclient.ResourceInterfaceFor(r)
		ctx, cancel := context.WithTimeoutCause(context.WithoutCancel(ctx), ct.FactorPlatformTimeout(1*time.Minute), errors.New("deletion took to long"))
		defer cancel()
		if err != nil {
			return err
		}
		if err := client.Delete(ctx, r.GetName(), metav1.DeleteOptions{
			PropagationPolicy: &bgDeletion,
		}); err != nil {
			return err
		}
	}

	return nil
}
