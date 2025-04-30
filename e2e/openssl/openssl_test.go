// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package openssl

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
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
)

// namespace the tests are executed in.
const (
	opensslFrontend = "openssl-frontend"
	opensslBackend  = "openssl-backend"

	meshCAFile = "mesh-ca.pem"
	rootCAFile = "coordinator-root-ca.pem"
)

// TestOpenSSL runs e2e tests on the example OpenSSL deployment.
func TestOpenSSL(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.OpenSSL()
	coordinator := kuberesource.CoordinatorBundle()

	resources = append(resources, coordinator...)

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	t.Run("check coordinator metrics and probe endpoints", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		require := require.New(t)

		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))

		frontendPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, opensslFrontend)
		require.NoError(err)
		require.Len(frontendPods, 1, "pod not found: %s/%s", ct.Namespace, opensslFrontend)

		coordinatorPods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "coordinator")
		require.NoError(err)
		require.NotEmpty(coordinatorPods, "pod not found: %s/%s", ct.Namespace, "coordinator")

		// deploy an additional port forwarder for metrics and probes
		stableNetworkID := fmt.Sprintf("%s.coordinator.%s.svc.cluster.local", coordinatorPods[0].Name, ct.Namespace)
		t.Logf("Constructed stable network ID %q", stableNetworkID)
		additionalForwarder := kuberesource.
			PortForwarder("coordinator-metrics", ct.Namespace).
			WithListenPorts([]int32{9102}).
			WithForwardTarget(stableNetworkID)

		patchedForwarder, err := kuberesource.ResourcesToUnstructured(kuberesource.PatchImages([]any{additionalForwarder.PodApplyConfiguration}, ct.ImageReplacements))
		require.NoError(err)
		require.NoError(ct.Kubeclient.Apply(ctx, patchedForwarder...))

		t.Cleanup(func() {
			_ = ct.Kubeclient.Client.CoreV1().Pods(ct.Namespace).Delete(context.Background(), "port-forwarder-coordinator-metrics", metav1.DeleteOptions{}) //nolint:usetesting, see https://github.com/ldez/usetesting/issues/4
		})

		argv := []string{"/bin/sh", "-c", "curl --fail " + net.JoinHostPort(coordinatorPods[0].Status.PodIP, "9102") + "/metrics"}
		_, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, frontendPods[0].Name, argv)
		require.NoError(err, "stderr: %q", stderr)

		for _, endpoint := range []string{"/probe/startup", "/probe/liveness", "/probe/readiness"} {
			require.NoError(ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-coordinator-metrics", "9102", func(addr string) error {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+endpoint, nil)
				if err != nil {
					return err
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("unexpected status code from probe %q: %d", endpoint, resp.StatusCode)
				}
				return nil
			}))
		}
	})

	for cert, pool := range map[string]*x509.CertPool{
		"mesh CA cert": ct.MeshCACert(),
		"root CA cert": ct.RootCACert(),
	} {
		t.Run("go dial frontend with "+cert, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
			defer cancel()

			require := require.New(t)

			require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))
			require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Pod{}, ct.Namespace, "port-forwarder-openssl-frontend"))

			require.NoError(ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-openssl-frontend", "443", func(addr string) error {
				dialer := &tls.Dialer{Config: &tls.Config{RootCAs: pool, ServerName: opensslFrontend}}
				conn, err := dialer.DialContext(ctx, "tcp", addr)
				if err == nil {
					conn.Close()
				}
				return err
			}))
		})
	}

	t.Run("certificates can be used by OpenSSL", func(t *testing.T) {
		// This test verifies that the certificates minted by the coordinator are accepted by OpenSSL in server and client mode.
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		c := kubeclient.NewForTest(t)

		require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, opensslBackend))

		// Call the backend server from the frontend. If this command produces no TLS error, we verified that
		// - the certificate in the frontend pod can be used as a client certificate
		// - the certificate in the backend pod can be used as a server certificate
		// - the backend's CA configuration accepted the frontend certificate
		// - the frontend's CA configuration accepted the backend certificate
		stdout, stderr, err := c.ExecDeployment(ctx, ct.Namespace, opensslFrontend, []string{"/bin/sh", "-c", opensslConnectCmd("openssl-backend:443", meshCAFile)})
		t.Log(stdout)
		require.NoError(err, "stderr: %q", stderr)
	})

	for _, deploymentToRestart := range []string{opensslBackend, opensslFrontend} {
		t.Run(fmt.Sprintf("certificate rotation and %s restart", deploymentToRestart), func(t *testing.T) {
			require := require.New(t)

			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
			defer cancel()

			c := kubeclient.NewForTest(t)

			// If in the future a SetManifest call with the same manifest does not result in a certificate rotation,
			// this change of the manifest makes sure to always rotate certificates.
			manifestBytes, err := os.ReadFile(ct.WorkDir + "/manifest.json")
			require.NoError(err)
			var m manifest.Manifest
			require.NoError(json.Unmarshal(manifestBytes, &m))
			// Add test domain name to first policy.
			for policyHash := range m.Policies {
				entry := m.Policies[policyHash]
				entry.SANs = append(entry.SANs, fmt.Sprintf("test-%s", deploymentToRestart))
				m.Policies[policyHash] = entry
				break
			}
			manifestBytes, err = json.Marshal(m)
			require.NoError(err)
			require.NoError(os.WriteFile(ct.WorkDir+"/manifest.json", manifestBytes, 0o644))

			// SetManifest rotates the certificates in the coordinator.
			ct.Set(t)

			// Restart one deployment so it has the new certificates.
			require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, deploymentToRestart))
			require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, deploymentToRestart))

			// This should not succeed because the certificates have changed.
			stdout, stderr, err := c.ExecDeployment(ctx, ct.Namespace, opensslFrontend, []string{"/bin/sh", "-c", opensslConnectCmd("openssl-backend:443", meshCAFile)})
			t.Log("openssl with wrong certificates:", stdout)
			require.Error(err)
			require.Contains(stderr, "self-signed certificate in certificate chain", "err: %s", err)

			// Connect from backend to fronted, because the frontend does not require client certs.
			// This should succeed because the root cert did not change.
			stdout, stderr, err = c.ExecDeployment(ctx, ct.Namespace, opensslBackend, []string{"/bin/sh", "-c", opensslConnectCmd("openssl-frontend:443", rootCAFile)})
			t.Log("openssl with root certificate:", stdout)
			require.NoError(err, "stderr: %q", stderr)

			// Restart the other deployment so both workloads have the same certificates.
			d := opensslBackend
			if deploymentToRestart == opensslBackend {
				d = opensslFrontend
			}
			require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, d))
			require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, d))

			// This should succeed since both workloads now have updated certificates.
			stdout, stderr, err = c.ExecDeployment(ctx, ct.Namespace, opensslFrontend, []string{"/bin/sh", "-c", opensslConnectCmd("openssl-backend:443", meshCAFile)})
			t.Log("openssl with correct certificates:", stdout)
			require.NoError(err, "stderr: %q", stderr)
		})
	}

	t.Run("coordinator recovery", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute) // Already long timeout, not using ct.FactorPlatformTimeout.
		defer cancel()

		c := kubeclient.NewForTest(t)

		require.NoError(c.Restart(ctx, kubeclient.StatefulSet{}, ct.Namespace, "coordinator"))
		require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.StatefulSet{}, ct.Namespace, "coordinator"))

		// TODO(freax13): The following verify sometimes fails spuriously due to
		//                connection issues. Waiting a little bit longer makes
		//                the whole test less flaky.
		time.Sleep(5 * time.Second)

		require.ErrorContains(ct.RunVerify(t.Context()), "recovery")

		require.True(t.Run("contrast recover", ct.Recover))

		require.True(t.Run("contrast verify", ct.Verify))

		require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))
		require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))

		t.Run("root CA is still accepted after coordinator recovery", func(t *testing.T) {
			stdout, stderr, err := c.ExecDeployment(ctx, ct.Namespace, opensslBackend, []string{"/bin/sh", "-c", opensslConnectCmd("openssl-frontend:443", rootCAFile)})
			if err != nil {
				t.Logf("openssl with %q after recovery:\n%s", rootCAFile, stdout)
			}
			assert.NoError(t, err, "stderr: %q", stderr)
		})

		t.Run("coordinator can't recover mesh CA key", func(t *testing.T) {
			_, _, err := c.ExecDeployment(ctx, ct.Namespace, opensslBackend, []string{"/bin/sh", "-c", opensslConnectCmd("openssl-frontend:443", meshCAFile)})
			assert.Error(t, err)
		})

		require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))
		require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, opensslBackend))

		t.Run("mesh CA after coordinator recovery is accepted when workloads are restarted", func(t *testing.T) {
			stdout, stderr, err := c.ExecDeployment(ctx, ct.Namespace, opensslBackend, []string{"/bin/sh", "-c", opensslConnectCmd("openssl-frontend:443", meshCAFile)})
			if err != nil {
				t.Logf("openssl with %q after recovery:\n%s", meshCAFile, stdout)
			}
			assert.NoError(t, err, "stderr: %q", stderr)
		})
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}

func opensslConnectCmd(addr, caCert string) string {
	return fmt.Sprintf(
		`openssl s_client -connect %s -verify_return_error -x509_strict -CAfile /contrast/tls-config/%s -cert /contrast/tls-config/certChain.pem -key /contrast/tls-config/key.pem </dev/null`,
		addr, caCert)
}
