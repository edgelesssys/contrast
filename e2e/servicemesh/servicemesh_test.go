//go:build e2e
// +build e2e

package servicemesh

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// namespace the tests are executed in.
const namespaceEnv = "K8S_NAMESPACE"

// TestIngress tests that the ingress proxies work as configured.
func TestIngress(t *testing.T) {
	c := kubeclient.NewForTest(t)

	namespace := os.Getenv(namespaceEnv)
	require.NotEmpty(t, namespace, "environment variable %q must be set", namespaceEnv)

	certs := make(map[string][]byte)

	require.True(t, t.Run("contrast verify", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		require := require.New(t)
		coordinator, cancelPortForward, err := c.PortForwardPod(ctx, namespace, "port-forwarder-coordinator", "1313")
		require.NoError(err)
		defer cancelPortForward()

		workspaceDir, err := os.MkdirTemp("", "nunki-verify.*")
		require.NoError(err)

		verify := cmd.NewVerifyCmd()
		verify.SetArgs([]string{
			"--workspace-dir", workspaceDir,
			"--coordinator-policy-hash=", // TODO(burgerdev): enable policy checking
			"--coordinator", coordinator,
		})
		verify.SetOut(io.Discard)
		errBuf := &bytes.Buffer{}
		verify.SetErr(errBuf)

		require.NoError(verify.Execute(), "could not verify coordinator: %s", errBuf)

		for _, certFile := range []string{
			"coordinator-root.pem",
			"mesh-root.pem",
		} {
			pem, err := os.ReadFile(path.Join(workspaceDir, certFile))
			assert.NoError(t, err)
			certs[certFile] = pem
		}
	}), "contrast verify needs to succeed for the following tests")

	for certFile, pem := range certs {
		t.Run("go dial web with ca "+certFile, func(t *testing.T) {
			require := require.New(t)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			web, cancelPortForward, err := c.PortForwardPod(ctx, namespace, "port-forwarder-emojivoto-web", "8080")
			require.NoError(err)
			t.Cleanup(cancelPortForward)

			pool := x509.NewCertPool()
			require.True(pool.AppendCertsFromPEM(pem))
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

		frontendPods, err := c.PodsFromDeployment(ctx, namespace, "web")
		require.NoError(err)
		require.Len(frontendPods, 1, "pod not found: %s/%s", namespace, "web")

		// The emoji service does not have an ingress proxy configuration, so we expect all ingress
		// traffic to be proxied with mandatory mutual TLS.
		// This test also verifies that client connections are not affected by the ingress proxy,
		// because we're running the commands on a pod with enabled proxy.

		argv := []string{"curl", "-sS", "--cacert", "/tls-config/MeshCACert.pem", "https://emoji:8801/metrics"}
		// curl does not like the wildcard cert and the service name does not match the deployment
		// name (i.e., the CN), so we tell curl to connect to expect the deployment name but
		// resolve the service name.
		argv = append(argv, "--connect-to", "emoji:8801:emoji-svc:8801")
		stdout, stderr, err := c.Exec(ctx, namespace, frontendPods[0].Name, argv)
		require.Error(err, "Expected call without client certificate to fail.\nstdout: %s\nstderr: %q", stdout, stderr)

		argv = append(argv, "--cert", "/tls-config/certChain.pem", "--key", "/tls-config/key.pem")
		stdout, stderr, err = c.Exec(ctx, namespace, frontendPods[0].Name, argv)
		require.NoError(err, "Expected call with client certificate to succeed.\nstdout: %s\nstderr: %q", stdout, stderr)
	})
}
