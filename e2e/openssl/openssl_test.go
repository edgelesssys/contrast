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
	"github.com/stretchr/testify/require"
)

// namespace the tests are executed in.
const (
	opensslFrontend = "openssl-frontend"
	opensslBackend  = "openssl-backend"
)

var imageReplacementsFile string

// TestOpenSSL runs e2e tests on the example OpenSSL deployment.
func TestOpenSSL(t *testing.T) {
	ct := contrasttest.New(t, imageReplacementsFile)

	resources := kuberesource.OpenSSL()

	coordinator := kuberesource.CoordinatorBundle()
	resources = append(resources, coordinator...)

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

		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, opensslFrontend))

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

			require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, opensslFrontend))

			addr, cancelPortForward, err := ct.Kubeclient.PortForwardPod(ctx, ct.Namespace, "port-forwarder-openssl-frontend", "443")
			require.NoError(err)
			defer cancelPortForward()

			dialer := &tls.Dialer{Config: &tls.Config{RootCAs: pool}}
			conn, err := dialer.DialContext(ctx, "tcp", addr)
			require.NoError(err)
			conn.Close()
		})
	}

	t.Run("certificates can be used by OpenSSL", func(t *testing.T) {
		// This test verifies that the certificates minted by the coordinator are accepted by OpenSSL in server and client mode.
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		c := kubeclient.NewForTest(t)

		require.NoError(c.WaitForDeployment(ctx, ct.Namespace, opensslBackend))

		// Call the backend server from the frontend. If this command produces no TLS error, we verified that
		// - the certificate in the frontend pod can be used as a client certificate
		// - the certificate in the backend pod can be used as a server certificate
		// - the backend's CA configuration accepted the frontend certificate
		// - the frontend's CA configuration accepted the backend certificate
		stdout, stderr, err := c.ExecDeployment(ctx, ct.Namespace, opensslFrontend, []string{"/bin/bash", "-c", opensslConnectCmd("openssl-backend:443", "mesh-ca.pem")})
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
				m.Policies[policyHash] = append(m.Policies[policyHash], fmt.Sprintf("test-%s", deploymentToRestart))
				break
			}
			manifestBytes, err = json.Marshal(m)
			require.NoError(err)
			require.NoError(os.WriteFile(ct.WorkDir+"/manifest.json", manifestBytes, 0o644))

			// SetManifest rotates the certificates in the coordinator.
			ct.Set(t)

			// Restart one deployment so it has the new certificates.
			require.NoError(c.RestartDeployment(ctx, ct.Namespace, deploymentToRestart))
			require.NoError(c.WaitForDeployment(ctx, ct.Namespace, deploymentToRestart))

			// This should not succeed because the certificates have changed.
			stdout, stderr, err := c.ExecDeployment(ctx, ct.Namespace, opensslFrontend, []string{"/bin/bash", "-c", opensslConnectCmd("openssl-backend:443", "mesh-ca.pem")})
			t.Log("openssl with wrong certificates:", stdout)
			require.Error(err)
			require.Contains(stderr, "self-signed certificate in certificate chain")

			// Connect from backend to fronted, because the frontend does not require client certs.
			// This should succeed because the root cert did not change.
			stdout, stderr, err = c.ExecDeployment(ctx, ct.Namespace, opensslBackend, []string{"/bin/bash", "-c", opensslConnectCmd("openssl-frontend:443", "coordinator-root-ca.pem")})
			t.Log("openssl with root certificate:", stdout)
			require.NoError(err, "stderr: %q", stderr)

			// Restart the other deployment so both workloads have the same certificates.
			d := opensslBackend
			if deploymentToRestart == opensslBackend {
				d = opensslFrontend
			}
			require.NoError(c.RestartDeployment(ctx, ct.Namespace, d))
			require.NoError(c.WaitForDeployment(ctx, ct.Namespace, d))

			// This should succeed since both workloads now have updated certificates.
			stdout, stderr, err = c.ExecDeployment(ctx, ct.Namespace, opensslFrontend, []string{"/bin/bash", "-c", opensslConnectCmd("openssl-backend:443", "mesh-ca.pem")})
			t.Log("openssl with correct certificates:", stdout)
			require.NoError(err, "stderr: %q", stderr)
		})
	}
}

func TestMain(m *testing.M) {
	flag.Parse()

	imageReplacementsFile = flag.Arg(0)

	os.Exit(m.Run())
}

func opensslConnectCmd(addr, caCert string) string {
	return fmt.Sprintf(
		`openssl s_client -connect %s -verify_return_error -x509_strict -CAfile /tls-config/%s -cert /tls-config/certChain.pem -key /tls-config/key.pem </dev/null`,
		addr, caCert)
}
