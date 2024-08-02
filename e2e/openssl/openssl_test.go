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
)

// namespace the tests are executed in.
const (
	opensslFrontend = "openssl-frontend"
	opensslBackend  = "openssl-backend"

	meshCAFile = "mesh-ca.pem"
	rootCAFile = "coordinator-root-ca.pem"
)

var (
	imageReplacementsFile, namespaceFile string
	skipUndeploy                         bool
)

// TestOpenSSL runs e2e tests on the example OpenSSL deployment.
func TestOpenSSL(t *testing.T) {
	ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, skipUndeploy)

	// TODO(msanft): Make this configurable
	platform := platforms.AKSCloudHypervisorSNP

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

	t.Run("check coordinator metrics endpoint", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		require := require.New(t)

		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))

		frontendPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, opensslFrontend)
		require.NoError(err)
		require.Len(frontendPods, 1, "pod not found: %s/%s", ct.Namespace, opensslFrontend)

		coordinatorPods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "coordinator")
		require.NoError(err)
		require.NotEmpty(coordinatorPods, "pod not found: %s/%s", ct.Namespace, "coordinator")

		argv := []string{"/bin/bash", "-c", "curl --fail " + net.JoinHostPort(coordinatorPods[0].Status.PodIP, "9102") + "/metrics"}
		_, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, frontendPods[0].Name, argv)
		require.NoError(err, "stderr: %q", stderr)
	})

	for cert, pool := range map[string]*x509.CertPool{
		"mesh CA cert": ct.MeshCACert(),
		"root CA cert": ct.RootCACert(),
	} {
		t.Run("go dial frontend with "+cert, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			require := require.New(t)

			require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))

			require.NoError(ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-openssl-frontend", "443", func(addr string) error {
				dialer := &tls.Dialer{Config: &tls.Config{RootCAs: pool}}
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

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		c := kubeclient.NewForTest(t)

		require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))

		// Call the backend server from the frontend. If this command produces no TLS error, we verified that
		// - the certificate in the frontend pod can be used as a client certificate
		// - the certificate in the backend pod can be used as a server certificate
		// - the backend's CA configuration accepted the frontend certificate
		// - the frontend's CA configuration accepted the backend certificate
		stdout, stderr, err := c.ExecDeployment(ctx, ct.Namespace, opensslFrontend, []string{"/bin/bash", "-c", opensslConnectCmd("openssl-backend:443", meshCAFile)})
		t.Log(stdout)
		require.NoError(err, "stderr: %q", stderr)
	})

	for _, deploymentToRestart := range []string{opensslBackend, opensslFrontend} {
		t.Run(fmt.Sprintf("certificate rotation and %s restart", deploymentToRestart), func(t *testing.T) {
			require := require.New(t)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
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
			require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, deploymentToRestart))

			// This should not succeed because the certificates have changed.
			stdout, stderr, err := c.ExecDeployment(ctx, ct.Namespace, opensslFrontend, []string{"/bin/bash", "-c", opensslConnectCmd("openssl-backend:443", meshCAFile)})
			t.Log("openssl with wrong certificates:", stdout)
			require.Error(err)
			require.Contains(stderr, "self-signed certificate in certificate chain")

			// Connect from backend to fronted, because the frontend does not require client certs.
			// This should succeed because the root cert did not change.
			stdout, stderr, err = c.ExecDeployment(ctx, ct.Namespace, opensslBackend, []string{"/bin/bash", "-c", opensslConnectCmd("openssl-frontend:443", rootCAFile)})
			t.Log("openssl with root certificate:", stdout)
			require.NoError(err, "stderr: %q", stderr)

			// Restart the other deployment so both workloads have the same certificates.
			d := opensslBackend
			if deploymentToRestart == opensslBackend {
				d = opensslFrontend
			}
			require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, d))
			require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, d))

			// This should succeed since both workloads now have updated certificates.
			stdout, stderr, err = c.ExecDeployment(ctx, ct.Namespace, opensslFrontend, []string{"/bin/bash", "-c", opensslConnectCmd("openssl-backend:443", meshCAFile)})
			t.Log("openssl with correct certificates:", stdout)
			require.NoError(err, "stderr: %q", stderr)
		})
	}

	t.Run("coordinator recovery", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		c := kubeclient.NewForTest(t)

		require.NoError(c.Restart(ctx, kubeclient.StatefulSet{}, ct.Namespace, "coordinator"))

		require.ErrorContains(ct.RunVerify(), "recovery")

		require.True(t.Run("contrast recover", ct.Recover))

		require.True(t.Run("contrast verify", ct.Verify))

		require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))
		require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))

		t.Run("root CA is still accepted after coordinator recovery", func(t *testing.T) {
			stdout, stderr, err := c.ExecDeployment(ctx, ct.Namespace, opensslBackend, []string{"/bin/bash", "-c", opensslConnectCmd("openssl-frontend:443", rootCAFile)})
			if err != nil {
				t.Logf("openssl with %q after recovery:\n%s", rootCAFile, stdout)
			}
			assert.NoError(t, err, "stderr: %q", stderr)
		})

		t.Run("coordinator can't recover mesh CA key", func(t *testing.T) {
			_, _, err := c.ExecDeployment(ctx, ct.Namespace, opensslBackend, []string{"/bin/bash", "-c", opensslConnectCmd("openssl-frontend:443", meshCAFile)})
			assert.Error(t, err)
		})

		require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))
		require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))

		t.Run("mesh CA after coordinator recovery is accepted when workloads are restarted", func(t *testing.T) {
			stdout, stderr, err := c.ExecDeployment(ctx, ct.Namespace, opensslBackend, []string{"/bin/bash", "-c", opensslConnectCmd("openssl-frontend:443", meshCAFile)})
			if err != nil {
				t.Logf("openssl with %q after recovery:\n%s", meshCAFile, stdout)
			}
			assert.NoError(t, err, "stderr: %q", stderr)
		})
	})
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.BoolVar(&skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}

func opensslConnectCmd(addr, caCert string) string {
	return fmt.Sprintf(
		`openssl s_client -connect %s -verify_return_error -x509_strict -CAfile /tls-config/%s -cert /tls-config/certChain.pem -key /tls-config/key.pem </dev/null`,
		addr, caCert)
}
