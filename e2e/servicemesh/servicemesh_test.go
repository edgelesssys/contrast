// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package servicemesh

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

var imageReplacements map[string]string

// TestIngressEgress tests that the ingress and egress proxies work as configured.
func TestIngressEgress(t *testing.T) {
	ct := contrasttest.New(t, imageReplacements)

	resources := kuberesource.Emojivoto(kuberesource.ServiceMeshIngressEgress)

	coordinator := kuberesource.CoordinatorBundle()
	resources = append(resources, coordinator...)

	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	require.True(t, t.Run("deployments become available", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, "vote-bot"))
		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, "emoji"))
		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, "voting"))
		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, "web"))
	}), "deployments need to be ready for subsequent tests")

	certs := map[string]*x509.CertPool{
		"coordinator-root.pem": ct.RootCACert(),
		"mesh-ca.pem":          ct.MeshCACert(),
	}
	for certFile, pool := range certs {
		t.Run("go dial web with ca "+certFile, func(t *testing.T) {
			require := require.New(t)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			web, cancelPortForward, err := ct.Kubeclient.PortForwardPod(ctx, ct.Namespace, "port-forwarder-web-svc", "443")
			require.NoError(err)
			t.Cleanup(cancelPortForward)

			tlsConf := &tls.Config{RootCAs: pool}
			hc := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConf}}
			req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/", web), nil)
			require.NoError(err)
			resp, err := hc.Do(req)
			require.NoError(err)
			defer resp.Body.Close()
			require.Equal(http.StatusOK, resp.StatusCode)
		})
	}

	t.Run("client certificates are required if not explicitly disabled", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		c := kubeclient.NewForTest(t)

		frontendPods, err := c.PodsFromDeployment(ctx, ct.Namespace, "web")
		require.NoError(err)
		require.Len(frontendPods, 1, "pod not found: %s/%s", ct.Namespace, "web")

		// The emoji service does not have an ingress proxy configuration, so we expect all ingress
		// traffic to be proxied with mandatory mutual TLS.
		// This test also verifies that client connections are not affected by the ingress proxy,
		// because we're running the commands on a pod with enabled proxy.

		argv := []string{"curl", "-sS", "--cacert", "/tls-config/mesh-ca.pem", "https://emoji:8801/metrics"}
		// curl does not like the wildcard cert and the service name does not match the deployment
		// name (i.e., the CN), so we tell curl to connect to expect the deployment name but
		// resolve the service name.
		argv = append(argv, "--connect-to", "emoji:8801:emoji-svc:8801")
		stdout, stderr, err := c.Exec(ctx, ct.Namespace, frontendPods[0].Name, argv)
		require.Error(err, "Expected call without client certificate to fail.\nstdout: %s\nstderr: %q", stdout, stderr)

		argv = append(argv, "--cert", "/tls-config/certChain.pem", "--key", "/tls-config/key.pem")
		stdout, stderr, err = c.Exec(ctx, ct.Namespace, frontendPods[0].Name, argv)
		require.NoError(err, "Expected call with client certificate to succeed.\nstdout: %s\nstderr: %q", stdout, stderr)
	})

	t.Run("admin interface is available", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		c := kubeclient.NewForTest(t)

		backendPods, err := c.PodsFromDeployment(ctx, ct.Namespace, "emoji")
		require.NoError(err)
		require.Len(backendPods, 1, "pod not found: %s/%s", ct.Namespace, "emoji")

		frontendPods, err := c.PodsFromDeployment(ctx, ct.Namespace, "web")
		require.NoError(err)
		require.Len(frontendPods, 1, "pod not found: %s/%s", ct.Namespace, "web")

		argv := []string{"curl", "-fsS", net.JoinHostPort(frontendPods[0].Status.PodIP, "9901") + "/stats/prometheus"}
		stdout, stderr, err := c.Exec(ctx, ct.Namespace, backendPods[0].Name, argv)
		require.NoError(err, "Expected Service Mesh admin interface to be reachable.\nstdout: %s\nstderr: %q", stdout, stderr)
	})
}

func TestMain(m *testing.M) {
	flag.Parse()

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("could not open image definition file %q: %v", flag.Arg(0), err)
	}
	imageReplacements, err = kuberesource.ImageReplacementsFromFile(f)
	if err != nil {
		log.Fatalf("could not parse image definition file %q: %v", flag.Arg(0), err)
	}

	os.Exit(m.Run())
}
