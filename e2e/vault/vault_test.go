// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package vault

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"

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

	// Overwrite the workloadSecretID in the manifest, to align with the value set in the
	// unsealing configuration of the Vault deployment.
	ct.PatchManifest(t, func(m manifest.Manifest) manifest.Manifest {
		for key, policy := range m.Policies {
			// TODO(jmxnzo): only do this for Vault, not the other deployment
			policy.WorkloadSecretID = "vault_unsealing"
			m.Policies[key] = policy
		}
		return m
	})

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	// Get Vault pod for subsequent tests.
	ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*60*time.Second))
	defer cancel()
	require.NoError(t, ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, "vault"))
	pods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "vault")
	require.NoError(t, err)
	require.Len(t, pods, 1)

	var token string

	t.Run("transit engine api auto-unseals vault", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(60*time.Second))
		defer cancel()

		// The bao operator init command should not be scripted like this in a real production environment,
		// because it will leak the recovery keys to anyone with kubernetes log access.
		// Normally this should be executed by an admin setting up the Vault once and the
		// recovery keys should be stored securely.
		stdOut, stdErr, err := ct.Kubeclient.ExecContainer(
			ctx,
			ct.Namespace,
			pods[0].Name,
			"openbao-client",
			[]string{
				"sh",
				"-c",
				"bao operator init",
			},
		)

		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
		token, err = extractVaultRootToken(stdOut)
		require.NoError(err, "failed to extract Vault root token from logs:\n%s", stdOut)
		require.NotEmpty(token, "extracted token is empty")

		// Implicitly verifies that Vault is automatically unsealed using the coordinator's transit engine API.
		sealed, err := checkSealingStatus(ctx, ct.Kubeclient, ct.Namespace, pods[0].Name, "openbao-client")
		require.NoError(err)
		require.False(sealed)
	})

	// Enables the KV secrets engine, and writes
	// a test secret to verify that Vault accepts and stores data correctly.
	// This secret will later be used to validate data persistence after a Vault restart.
	t.Run("enable KV engine and create secret on vault", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(60*time.Second))
		defer cancel()

		stdOut, stdErr, err := ct.Kubeclient.ExecContainer(
			ctx,
			ct.Namespace,
			pods[0].Name,
			"openbao-client",
			[]string{
				"sh",
				"-c",
				fmt.Sprintf("VAULT_TOKEN=%s bao secrets enable -path=mykv kv", token),
			},
		)

		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)

		stdOut, stdErr, err = ct.Kubeclient.ExecContainer(
			ctx,
			ct.Namespace,
			pods[0].Name,
			"openbao-client",
			[]string{
				"sh",
				"-c",
				fmt.Sprintf("VAULT_TOKEN=%s bao kv put mykv/hello foo=bar", token),
			},
		)
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)

		stdOut, stdErr, err = ct.Kubeclient.ExecContainer(
			ctx,
			ct.Namespace,
			pods[0].Name,
			"openbao-client",
			[]string{
				"sh", "-c",
				fmt.Sprintf("VAULT_TOKEN=%s bao kv get mykv/hello", token),
			},
		)
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

		stdOut, stdErr, err := ct.Kubeclient.ExecContainer(
			ctx,
			ct.Namespace,
			pods[0].Name,
			"openbao-client",
			[]string{
				"sh",
				"-c",
				fmt.Sprintf("VAULT_TOKEN=%s bao operator seal", token),
			},
		)
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
	})

	// Restarts the Vault deployment and verifies it auto-unseals using the transit engine API.
	// Additionally confirms that the previously written secret still exists, ensuring data persistence.
	t.Run("vault auto-unseals after restart and secrets are persistent", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(60*time.Second))
		defer cancel()

		require.NoError(ct.Kubeclient.Restart(ctx, kubeclient.StatefulSet{}, ct.Namespace, "vault"))
		require.NoError(ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, "vault"))

		pods, err = ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "vault")
		require.NoError(err)
		require.Len(pods, 1)

		// Implicitly verifies that Vault is automatically unsealed using the coordinator's transit engine API.0
		sealed, err := checkSealingStatus(ctx, ct.Kubeclient, ct.Namespace, pods[0].Name, "openbao-client")
		require.NoError(err)
		require.False(sealed)

		stdOut, stdErr, err := ct.Kubeclient.ExecContainer(
			ctx,
			ct.Namespace,
			pods[0].Name,
			"openbao-client",
			[]string{
				"sh", "-c",
				fmt.Sprintf("VAULT_TOKEN=%s bao kv get mykv/hello", token),
			},
		)
		require.NoError(err, "stderr: %s", stdErr)
		require.Contains(stdOut, "foo")
		require.Contains(stdOut, "bar")
	})
}

// checkSealingStatus checks the sealing status of the Vault by sending a request to bao status and returns true if Vault is sealed.
func checkSealingStatus(ctx context.Context, kubeclient *kubeclient.Kubeclient, namespace, podName, execContainerName string) (bool, error) {
	stdOut, stdErr, err := kubeclient.ExecContainer(ctx, namespace, podName, execContainerName, []string{"bao", "status"})
	if err != nil {
		return false, fmt.Errorf("error executing bao status: %w\nstderr: %s", err, stdErr)
	}

	for _, line := range strings.Split(stdOut, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Sealed") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1] == "true", nil
			}
		}
	}

	return false, fmt.Errorf("unable to determine seal status from output:\n%s", stdOut)
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

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
