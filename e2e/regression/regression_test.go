// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package regression

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/elazarl/goproxy"
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

			yamlDir := dataDir + dir.Name() + "/"
			files, err := os.ReadDir(yamlDir)
			require.NoError(err)

			containsCronJob := false
			intermediateResources := resources
			for _, file := range files {
				resourceYAML, err := os.ReadFile(yamlDir + file.Name())
				require.NoError(err)
				resourceYAML = bytes.ReplaceAll(resourceYAML, []byte("@@REPLACE_NAMESPACE@@"), []byte(ct.Namespace))

				newResources, err := kuberesource.UnmarshalApplyConfigurations(resourceYAML)
				require.NoError(err)

				newResources = kuberesource.PatchRuntimeHandlers(newResources, runtimeHandler)
				newResources = kuberesource.AddPortForwarders(newResources)

				intermediateResources = append(intermediateResources, newResources...)

				// Check if we are testing a cron job
				unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
				require.NoError(err)
				for _, resource := range unstructuredResources {
					if resource.GetKind() == "CronJob" {
						containsCronJob = true
					}
				}
			}
			// write the new resources.yml
			resourceBytes, err := kuberesource.EncodeResources(intermediateResources...)
			require.NoError(err)
			require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yml"), resourceBytes, 0o644))

			// generate, set, deploy and verify the new policy
			require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

			if containsCronJob {
				return
			}

			require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
			require.True(t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
			require.True(t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

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
					require.NoError(c.WaitForDeployment(ctx, ct.Namespace, *r.Name))
				case *applyappsv1.DaemonSetApplyConfiguration:
					require.NoError(c.WaitForDaemonSet(ctx, ct.Namespace, *r.Name))
				case *applycorev1.PodApplyConfiguration:
					require.NoError(c.WaitForPod(ctx, ct.Namespace, *r.Name))
				case *applybatchv1.JobApplyConfiguration:
					require.NoError(c.WaitForJob(ctx, ct.Namespace, *r.Name))
				case *applyappsv1.ReplicaSetApplyConfiguration:
					require.NoError(c.WaitForReplicaSet(ctx, ct.Namespace, *r.Name))
				case *applycorev1.ReplicationControllerApplyConfiguration:
					require.NoError(c.WaitForReplicationController(ctx, ct.Namespace, *r.Name))
				}
			}
		})
	}

	t.Run("http-proxy", func(t *testing.T) { testHTTPProxy(t, ct) })
}

func testHTTPProxy(t *testing.T, ct *contrasttest.ContrastTest) {
	// Start a proxy server

	proxy := goproxy.NewProxyHttpServer()
	server := http.Server{Handler: proxy}
	errCh := make(chan error)

	// coordinatorConnectionProxied will be set to true if the proxy performs an HTTP CONNECT to the address of the Coordinator.
	var coordinatorConnectionProxied atomic.Bool
	proxy.ConnectDial = func(network string, addr string) (net.Conn, error) {
		if strings.HasPrefix(addr, "0.0.0.0:") { // we use this address in ContrastTest.runAgainstCoordinator
			coordinatorConnectionProxied.Store(true)
		}
		return net.Dial(network, addr)
	}

	proxyListener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	proxyAddr := proxyListener.Addr().String()
	const invalidAddr = "127.0.0.1:0"

	t.Cleanup(func() {
		require.NoError(t, server.Close())
		err := <-errCh
		require.ErrorIs(t, err, http.ErrServerClosed)
	})

	go func() {
		errCh <- server.Serve(proxyListener)
	}()

	testCases := map[string]struct {
		env         map[string]string
		wantProxied bool
		wantErrMsg  string
	}{
		"proxy env not set": {
			wantProxied: false,
		},
		"https_proxy valid": {
			env:         map[string]string{"https_proxy": proxyAddr},
			wantProxied: true,
		},
		"https_proxy invalid": {
			env:        map[string]string{"https_proxy": invalidAddr},
			wantErrMsg: "transport: Error while dialing: dial tcp " + invalidAddr + ": connect: connection refused",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			if tc.wantErrMsg != "" {
				// only try verify because set uses a retry loop
				assert.ErrorContains(runCommand(ct, "verify"), tc.wantErrMsg)
				return
			}

			require.NoError(runCommand(ct, "set"))
			assert.Equal(tc.wantProxied, coordinatorConnectionProxied.Swap(false))

			require.NoError(runCommand(ct, "verify"))
			assert.Equal(tc.wantProxied, coordinatorConnectionProxied.Swap(false))
		})
	}
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	runCommandStr := flag.String("run-command", "", "")
	runNamespace := flag.String("run-namespace", "", "")
	runWorkdir := flag.String("run-workdir", "", "")
	flag.Parse()

	ctx := context.Background()

	if *runCommandStr != "" {
		if err := runCommandImpl(ctx, *runCommandStr, *runNamespace, *runWorkdir); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	os.Exit(m.Run())
}

// runCommand runs a CLI command in a new process so that the proxy env vars are re-read.
// Go caches the env vars, so we can't run the commands in the same process as usual.
func runCommand(ct *contrasttest.ContrastTest, cmd string) error {
	out, err := exec.Command(os.Args[0], "-run-command="+cmd, "-run-namespace="+ct.Namespace, "-run-workdir="+ct.WorkDir).CombinedOutput()
	if err != nil {
		return errors.New(string(out))
	}
	return nil
}

func runCommandImpl(ctx context.Context, cmd, namespace, workDir string) error {
	kclient, err := kubeclient.NewForTestWithoutT()
	if err != nil {
		return err
	}
	ct := &contrasttest.ContrastTest{
		Namespace:  namespace,
		WorkDir:    workDir,
		Kubeclient: kclient,
	}

	switch cmd {
	case "set":
		return ct.RunSet(ctx)
	case "verify":
		return ct.RunVerify(ctx)
	}

	return errors.New("unknown command: " + cmd)
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
