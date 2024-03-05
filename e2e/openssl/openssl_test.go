//go:build e2e
// +build e2e

package openssl

import (
	"context"
	"crypto/tls"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
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
	require := require.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	c := kubeclient.NewForTest(t)

	namespace := os.Getenv(namespaceEnv)
	require.NotEmpty(namespace, "environment variable %q must be set", namespaceEnv)

	addr, cancelPortForward, err := c.PortForwardPod(ctx, namespace, "port-forwarder-openssl-frontend", "443")
	require.NoError(err)
	defer cancelPortForward()

	// TODO(burgerdev): properly test chain to mesh root
	dialer := &tls.Dialer{Config: &tls.Config{InsecureSkipVerify: true}}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	require.NoError(err)
	tlsConn := conn.(*tls.Conn)

	var names []string
	for _, cert := range tlsConn.ConnectionState().PeerCertificates {
		names = append(names, cert.Subject.CommonName)
	}
	require.Contains(names, "openssl-frontend")
}
