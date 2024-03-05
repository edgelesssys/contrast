//go:build e2e
// +build e2e

package openssl

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
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

// TestBackend verifies that the certificates minted by the coordinator are accepted by OpenSSL in server and client mode.
//
// The test expects deployments/openssl to be available in the cluster (manifest set and workloads ready).
func TestFrontend2Backend(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	c := kubeclient.NewForTest(t)

	namespace := os.Getenv(namespaceEnv)
	require.NotEmpty(namespace, "environment variable %q must be set", namespaceEnv)

	frontendPods, err := c.PodsFromDeployment(ctx, namespace, "openssl-frontend")
	require.NoError(err)
	require.Len(frontendPods, 1, "pod not found: %s/%s", namespace, "openssl-frontend")

	// Call the backend server from the frontend. If this command produces no TLS error, we verified that
	// - the certificate in the frontend pod can be used as a client certificate
	// - the certificate in the backend pod can be used as a server certificate
	// - the backend's CA configuration accepted the frontend certificate
	// - the frontend's CA configuration accepted the backend certificate
	stdout, stderr, err := c.Exec(ctx, namespace, frontendPods[0].Name,
		[]string{"/bin/bash", "-c", `printf "GET / HTTP/1.0\nHost: openssl-backend\n" | openssl s_client -connect openssl-backend:443 -verify_return_error -CAfile /tls-config/MeshCACert.pem -cert /tls-config/certChain.pem -key /tls-config/key.pem`},
	)
	t.Log(stdout)
	require.NoError(err, "stderr: %q", stderr)
}

// TestFrontend verifies the certificate used by the OpenSSL frontend comes from the coordinator.
func TestFrontend(t *testing.T) {
	c := kubeclient.NewForTest(t)

	namespace := os.Getenv(namespaceEnv)
	require.NotEmpty(t, namespace, "environment variable %q must be set", namespaceEnv)

	certs := make(map[string][]byte)

	t.Run("contrast verify", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		require := require.New(t)
		coordinator, cancelPortForward, err := c.PortForwardPod(ctx, namespace, "port-forwarder-coordinator", "1313")
		require.NoError(err)
		defer cancelPortForward()

		output, err := os.MkdirTemp("", "nunki-verify.*")
		require.NoError(err)

		verify := cmd.NewVerifyCmd()
		verify.SetArgs([]string{
			"--output", output,
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
			pem, err := os.ReadFile(path.Join(output, certFile))
			assert.NoError(t, err)
			certs[certFile] = pem
		}
	})

	for certFile, pem := range certs {
		t.Run("go dial frontend with ca "+certFile, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			require := require.New(t)

			addr, cancelPortForward, err := c.PortForwardPod(ctx, namespace, "port-forwarder-openssl-frontend", "443")
			require.NoError(err)
			defer cancelPortForward()

			pool := x509.NewCertPool()
			require.True(pool.AppendCertsFromPEM(pem))
			dialer := &tls.Dialer{Config: &tls.Config{RootCAs: pool}}
			conn, err := dialer.DialContext(ctx, "tcp", addr)
			require.NoError(err)
			conn.Close()
		})
	}
}
