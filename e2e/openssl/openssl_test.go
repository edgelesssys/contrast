//go:build e2e
// +build e2e

package openssl

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/nunki/e2e/internal/kubeclient"
	"github.com/stretchr/testify/require"
)

// namespace the tests are executed in.
const namespaceEnv = "K8S_NAMESPACE"

// TestOpenssl verifies that the certificates minted by the coordinator are accepted by OpenSSL in server and client mode.
//
// The test expects deployments/openssl to be available in the cluster (manifest set and workloads ready).
func TestOpenSSL(t *testing.T) {
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
