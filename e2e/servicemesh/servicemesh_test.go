// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package servicemesh

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIngressEgress tests that the ingress and egress proxies work as configured.
func TestIngressEgress(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.Emojivoto(kuberesource.ServiceMeshIngressEgress)

	coordinator := kuberesource.CoordinatorBundle()

	resources = append(resources, coordinator...)

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	require.True(t, t.Run("deployments become available", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
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

			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
			defer cancel()

			require.NoError(ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-web-svc", "443", func(addr string) error {
				tlsConf := &tls.Config{RootCAs: pool, ServerName: "web"}
				hc := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConf}}
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://%s/", addr), http.NoBody)
				if !assert.NoError(t, err) {
					return nil
				}
				resp, err := hc.Do(req)
				if err != nil {
					return err
				}
				resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				return nil
			}))
		})
	}

	t.Run("client certificates are required if not explicitly disabled", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		c := kubeclient.NewForTest(t)

		frontendPods, err := c.PodsFromDeployment(ctx, ct.Namespace, "web")
		require.NoError(err)
		require.Len(frontendPods, 1, "pod not found: %s/%s", ct.Namespace, "web")

		emojiConnection := net.JoinHostPort(c.FirstPodIP(ctx, t, ct.Namespace, "emoji"), "8801")

		// The emoji service does not have an ingress proxy configuration, so we expect all ingress
		// traffic to be proxied with mandatory mutual TLS.
		// This test also verifies that client connections are not affected by the ingress proxy,
		// because we're running the commands on a pod with enabled proxy.

		argv := []string{"curl", "-sS", "--cacert", "/contrast/tls-config/mesh-ca.pem", fmt.Sprintf("https://%s/metrics", emojiConnection)}
		stdout, stderr, err := c.Exec(ctx, ct.Namespace, frontendPods[0].Name, argv)
		require.Error(err, "Expected call without client certificate to fail.\nstdout: %s\nstderr: %q", stdout, stderr)

		argv = append(argv, "--cert", "/contrast/tls-config/certChain.pem", "--key", "/contrast/tls-config/key.pem")
		stdout, stderr, err = c.Exec(ctx, ct.Namespace, frontendPods[0].Name, argv)
		require.NoError(err, "Expected call with client certificate to succeed.\nstdout: %s\nstderr: %q", stdout, stderr)
	})

	t.Run("admin interface is available", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		c := kubeclient.NewForTest(t)

		frontendPods, err := c.PodsFromDeployment(ctx, ct.Namespace, "voting")
		require.NoError(err)
		require.Len(frontendPods, 1, "pod not found: %s/%s", ct.Namespace, "voting")

		argv := []string{"curl", "-fsS", net.JoinHostPort(c.FirstPodIP(ctx, t, ct.Namespace, "emoji"), "9901") + "/stats/prometheus"}
		stdout, stderr, err := c.Exec(ctx, ct.Namespace, frontendPods[0].Name, argv)
		require.NoError(err, "Expected Service Mesh admin interface to be reachable.\nstdout: %s\nstderr: %q", stdout, stderr)
	})

	t.Run("voting works end-to-end", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		c := kubeclient.NewForTest(t)

		// Verify that outbound traffic is proxied as expected by checking the number of total votes in the metrics.

		emojiPods, err := c.PodsFromDeployment(ctx, ct.Namespace, "emoji")
		require.NoError(err)
		require.Len(emojiPods, 1, "pod not found: %s/%s", ct.Namespace, "emoji")

		ticker := time.NewTicker(2 * time.Second)

		for {
			select {
			case <-ctx.Done():
				require.Fail("No successful grpc calls reported before context expired", ctx.Err())
			case <-ticker.C:
			}
			stdout, stderr, err := c.Exec(ctx, ct.Namespace, emojiPods[0].Name, []string{"curl", "-fsS", "localhost:8801/metrics"})
			if err != nil {
				t.Logf("Could not query grpc metrics: %v\nstderr:\n%s", err, stderr)
				continue
			}

			parser := expfmt.NewTextParser(model.LegacyValidation)
			metrics, err := parser.TextToMetricFamilies(strings.NewReader(stdout))
			require.NoError(err, "Reply from /metrics endpoint did not parse as metrics")

			const metricName = "grpc_server_handled_total"
			metricFamily, ok := metrics[metricName]
			if !ok {
				t.Log("grpc metrics not yet available")
				continue
			}

			total := 0.0
			for _, metric := range metricFamily.GetMetric() {
				for _, labelPair := range metric.GetLabel() {
					if labelPair.Name == nil || *labelPair.Name != "grpc_code" {
						continue
					}
					if labelPair.Value == nil || *labelPair.Value != "OK" {
						break
					}
					total += metric.GetCounter().GetValue()
				}
			}
			if total > 0 {
				break
			}
			t.Logf("No successful grpc calls reported yet.")
		}
	})

	t.Run("egress without ingress fails closed", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		c := kubeclient.NewForTest(t)

		votingPods, err := c.PodsFromDeployment(ctx, ct.Namespace, "voting")
		require.NoError(err)
		require.Len(votingPods, 1, "pod not found: %s/%s", ct.Namespace, "voting")

		stdout, stderr, err := c.ExecContainer(ctx, ct.Namespace, votingPods[0].Name, "contrast-service-mesh", []string{"sh", "-ec", "iptables-save; iptables-legacy-save"})
		require.NoError(err, "Could not dump iptables.\nstdout: %s\nstderr: %q", stdout, stderr)
		require.Empty(stderr)
		require.Contains(stdout, "-j TPROXY")
	})

	t.Run("ingress can be disabled explicitly", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		c := kubeclient.NewForTest(t)

		voteBotPods, err := c.PodsFromDeployment(ctx, ct.Namespace, "vote-bot")
		require.NoError(err)
		require.Len(voteBotPods, 1, "pod not found: %s/%s", ct.Namespace, "vote-bot")

		stdout, stderr, err := c.ExecContainer(ctx, ct.Namespace, voteBotPods[0].Name, "contrast-service-mesh", []string{"sh", "-ec", "iptables-save; iptables-legacy-save"})
		require.NoError(err, "Could not dump iptables.\nstdout: %s\nstderr: %q", stdout, stderr)
		require.Empty(stderr)
		require.NotContains(stdout, "-j TPROXY")
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
