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
	"path/filepath"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kubeapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// namespace the tests are executed in.
const (
	namespaceEnv = "K8S_NAMESPACE"

	opensslFrontend = "openssl-frontend"
	opensslBackend  = "openssl-backend"
)

// TestOpenSSL runs e2e tests on the example OpenSSL deployment.
func TestOpenSSL(t *testing.T) {
	c := kubeclient.NewForTest(t)

	namespace := os.Getenv(namespaceEnv)
	require.NotEmpty(t, namespace, "environment variable %q must be set", namespaceEnv)

	resources, err := filepath.Glob("./workspace/deployment/*.yml")
	require.NoError(t, err)

	require.True(t, t.Run("generate", func(t *testing.T) {
		require := require.New(t)

		args := []string{
			"--workspace-dir", "./workspace",
		}
		args = append(args, resources...)

		generate := cmd.NewGenerateCmd()
		generate.Flags().String("workspace-dir", "", "") // Make generate aware of root flags
		generate.SetArgs(args)
		generate.SetOut(io.Discard)
		errBuf := &bytes.Buffer{}
		generate.SetErr(errBuf)

		require.NoError(generate.Execute(), "could not generate manifest: %s", errBuf)
	}))

	// TODO(burgerdev): policy hash should come from contrast generate output.
	coordinatorPolicyHashBytes, err := os.ReadFile("workspace/coordinator-policy.sha256")
	require.NoError(t, err)
	coordinatorPolicyHash := string(coordinatorPolicyHashBytes)
	require.NotEmpty(t, coordinatorPolicyHash, "expected apply to fill coordinator policy hash")

	require.True(t, t.Run("apply", func(t *testing.T) {
		require := require.New(t)

		var objects []*unstructured.Unstructured
		for _, file := range resources {
			yaml, err := os.ReadFile(file)
			require.NoError(err)
			fileObjects, err := kubeapi.UnmarshalUnstructuredK8SResource(yaml)
			require.NoError(err)
			objects = append(objects, fileObjects...)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		c := kubeclient.NewForTest(t)
		require.NoError(c.Apply(ctx, objects...))
	}), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		require.NoError(c.WaitForDeployment(ctx, namespace, "coordinator"))

		coordinator, cancelPortForward, err := c.PortForwardPod(ctx, namespace, "port-forwarder-coordinator", "1313")
		require.NoError(err)
		defer cancelPortForward()

		args := []string{
			"--coordinator-policy-hash", coordinatorPolicyHash,
			"--coordinator", coordinator,
			"--workspace-dir", "./workspace",
		}
		args = append(args, resources...)

		set := cmd.NewSetCmd()
		set.Flags().String("workspace-dir", "", "") // Make set aware of root flags
		set.SetArgs(args)
		set.SetOut(io.Discard)
		errBuf := &bytes.Buffer{}
		set.SetErr(errBuf)

		require.NoError(set.Execute(), "could not set manifest at coordinator: %s", errBuf)
	}), "contrast set needs to succeed for subsequent tests")

	certs := make(map[string][]byte)

	require.True(t, t.Run("contrast verify", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		require.NoError(c.WaitForDeployment(ctx, namespace, "coordinator"))

		coordinator, cancelPortForward, err := c.PortForwardPod(ctx, namespace, "port-forwarder-coordinator", "1313")
		require.NoError(err)
		defer cancelPortForward()

		workspaceDir, err := os.MkdirTemp("", "contrast-verify.*")
		require.NoError(err)

		verify := cmd.NewVerifyCmd()
		verify.SetArgs([]string{
			"--workspace-dir", workspaceDir,
			"--coordinator-policy-hash", coordinatorPolicyHash,
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
	}), "contrast verify needs to succeed for subsequent tests")

	for certFile, pem := range certs {
		t.Run("go dial frontend with ca "+certFile, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			require := require.New(t)

			require.NoError(c.WaitForDeployment(ctx, namespace, opensslFrontend))

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

	// TODO(burgerdev): this test should be run with its own kubectl apply/contrast set preface.
	t.Run("certificates can be used by OpenSSL", func(t *testing.T) {
		// This test verifies that the certificates minted by the coordinator are accepted by OpenSSL in server and client mode.
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		c := kubeclient.NewForTest(t)

		require.NoError(c.WaitForDeployment(ctx, namespace, opensslFrontend))
		require.NoError(c.WaitForDeployment(ctx, namespace, opensslBackend))

		frontendPods, err := c.PodsFromDeployment(ctx, namespace, opensslFrontend)
		require.NoError(err)
		require.Len(frontendPods, 1, "pod not found: %s/%s", namespace, opensslFrontend)

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
	})
}
