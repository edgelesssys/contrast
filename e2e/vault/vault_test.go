// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package vault

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
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

func TestVault(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.Vault(ct.Namespace)

	coordinator := kuberesource.CoordinatorBundle()

	resources = append(resources, coordinator...)

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	// Get Vault pod for subsequent tests.
	ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
	defer cancel()
	require.NoError(t, ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, "vault"))
	pods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "vault")
	require.NoError(t, err)
	require.Len(t, pods, 1)

	var token string

	t.Run("transit engine api auto-unseals vault", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(time.Minute))
		defer cancel()

		// The bao operator init command should not be scripted like this in a real production environment,
		// because it will leak the recovery keys to anyone with kubernetes log access.
		// Normally this should be executed by an admin setting up the Vault once and the
		// recovery keys should be stored securely.
		// TODO(burgerdev): init and configuration should be performed with remote requests instead of exec.
		stdOut, stdErr, err := ct.Kubeclient.ExecContainer(
			ctx,
			ct.Namespace,
			pods[0].Name,
			"openbao-server",
			[]string{
				"sh",
				"-c",
				"VAULT_CACERT=/contrast/tls-config/mesh-ca.pem VAULT_ADDR=https://vault:8200 bao operator init",
			},
		)

		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
		token, err = extractVaultRootToken(stdOut)
		require.NoError(err, "failed to extract Vault root token from logs:\n%s", stdOut)
	})
	require.NotEmpty(t, token, "need a root token for subsequent tests")

	runVaultScript := func(ctx context.Context, script string) (string, string, error) {
		return ct.Kubeclient.ExecContainer(
			ctx,
			ct.Namespace,
			pods[0].Name,
			"openbao-server",
			[]string{
				"sh",
				"-c",
				fmt.Sprintf("export VAULT_TOKEN=%s; export VAULT_CACERT=/contrast/tls-config/mesh-ca.pem; export VAULT_ADDR=https://vault:8200; %s", token, script),
			},
		)
	}

	// Enables the KV secrets engine, and writes
	// a test secret to verify that Vault accepts and stores data correctly.
	// This secret will later be used to validate data persistence after a Vault restart.
	t.Run("enable KV engine and create secret on vault", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(time.Minute))
		defer cancel()

		stdOut, stdErr, err := runVaultScript(ctx, "bao secrets enable kv")
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)

		stdOut, stdErr, err = runVaultScript(ctx, "bao kv put kv/hello foo=bar")
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)

		stdOut, stdErr, err = runVaultScript(ctx, "bao kv get kv/hello")
		require.NoError(err, "stderr: %s", stdErr)
		require.Contains(stdOut, "foo")
		require.Contains(stdOut, "bar")
	})

	// Reuses the previously extracted root token to manually seal the Vault instance.
	// Verifies that the seal command completes successfully.
	t.Run("vault can be sealed manually", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(60*time.Second))
		defer cancel()

		stdOut, stdErr, err := runVaultScript(ctx, "bao operator seal")
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
	})

	// Restarts the Vault deployment and verifies it auto-unseals using the transit engine API.
	// Additionally confirms that the previously written secret still exists, ensuring data persistence.
	t.Run("vault auto-unseals after restart and secrets are persistent", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		defer cancel()

		require.NoError(ct.Kubeclient.Restart(ctx, kubeclient.StatefulSet{}, ct.Namespace, "vault"))
		require.NoError(ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, "vault"))

		require.EventuallyWithT(func(t *assert.CollectT) {
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			assert.NoError(t, ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-vault", "8200", isVaultUnsealed))
		}, time.Minute, 5*time.Second)

		pods, err = ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "vault")
		require.NoError(err)
		require.Len(pods, 1)

		stdOut, stdErr, err := runVaultScript(ctx, "bao kv get kv/hello")
		require.NoError(err, "stderr: %s", stdErr)
		require.Contains(stdOut, "foo")
		require.Contains(stdOut, "bar")
	})

	// Configures a policy that allows Contrast workloads to access the KV secret, and checks
	// whether that policy actually works.
	t.Run("vault policy can be defined for Contrast workloads", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(60*time.Second))
		defer cancel()

		stdOut, stdErr, err := runVaultScript(ctx, `
bao auth enable cert
bao write auth/cert/certs/coordinator display_name=coordinator policies=contrast certificate=@/contrast/tls-config/coordinator-root-ca.pem
bao policy write contrast - <<EOF
path "kv/*"
{
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF
		`)
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)

		// The vault-client is configured with a readiness probe that succeeds when it can login
		// using Contrast certs and retrieve the secret.
		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, "vault-client"))
	})
}

// extractVaultRootToken parses Vault logs and extracts the initial root token.
// It searches for a line starting with "Initial Root Token:" and returns the token value.
// Returns an error if the token is not found in the provided log output.
func extractVaultRootToken(logs string) (string, error) {
	re := regexp.MustCompile(`(?m)^Initial Root Token:\s+(\S+)`)
	matches := re.FindStringSubmatch(logs)
	if len(matches) < 2 {
		return "", fmt.Errorf("root token not found")
	}
	return matches[1], nil
}

type vaultSealStatus struct {
	Initialized *bool `json:"initialized"`
	Sealed      *bool `json:"sealed"`
}

func (s vaultSealStatus) unsealed() bool {
	initialized := s.Initialized != nil && *s.Initialized
	sealed := s.Sealed != nil && *s.Sealed
	return initialized && !sealed
}

func isVaultUnsealed(addr string) error {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	resp, err := client.Get(fmt.Sprintf("https://%s/v1/sys/seal-status", addr)) //nolint:noctx // Context is implicitly enforced by the port forwarder.
	if err != nil {
		return fmt.Errorf("getting seal status: %w", err)
	}
	defer resp.Body.Close()

	var status vaultSealStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("decoding seal status response: %w", err)
	}
	if !status.unsealed() {
		return fmt.Errorf("vault status: %#v", status)
	}
	return nil
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
