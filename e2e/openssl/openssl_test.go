//go:build e2e
// +build e2e

package openssl

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/e2e/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

// namespace the tests are executed in.
const (
	opensslFrontend = "openssl-frontend"
	opensslBackend  = "openssl-backend"
)

var imageReplacements map[string]string

// TestOpenSSL runs e2e tests on the example OpenSSL deployment.
func TestOpenSSL(t *testing.T) {
	ct := contrasttest.New(t, imageReplacements)

	resources, err := kuberesource.OpenSSL()
	require.NoError(t, err)

	ct.Init(t, resources)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

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

	// TODO(burgerdev): this test should be run with its own kubectl apply/contrast set preface.
	t.Run("certificates can be used by OpenSSL", func(t *testing.T) {
		// This test verifies that the certificates minted by the coordinator are accepted by OpenSSL in server and client mode.
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		c := kubeclient.NewForTest(t)

		require.NoError(c.WaitForDeployment(ctx, ct.Namespace, opensslFrontend))
		require.NoError(c.WaitForDeployment(ctx, ct.Namespace, opensslBackend))

		frontendPods, err := c.PodsFromDeployment(ctx, ct.Namespace, opensslFrontend)
		require.NoError(err)
		require.Len(frontendPods, 1, "pod not found: %s/%s", ct.Namespace, opensslFrontend)

		// Call the backend server from the frontend. If this command produces no TLS error, we verified that
		// - the certificate in the frontend pod can be used as a client certificate
		// - the certificate in the backend pod can be used as a server certificate
		// - the backend's CA configuration accepted the frontend certificate
		// - the frontend's CA configuration accepted the backend certificate
		stdout, stderr, err := c.Exec(ctx, ct.Namespace, frontendPods[0].Name,
			[]string{"/bin/bash", "-c", `printf "GET / HTTP/1.0\nHost: openssl-backend\n" | openssl s_client -connect openssl-backend:443 -verify_return_error -CAfile /tls-config/MeshCACert.pem -cert /tls-config/certChain.pem -key /tls-config/key.pem`},
		)
		t.Log(stdout)
		require.NoError(err, "stderr: %q", stderr)
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
