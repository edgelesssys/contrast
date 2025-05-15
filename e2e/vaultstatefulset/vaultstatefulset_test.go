// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package vaultstatefulset

import (
	"bufio"
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

func TestVaultStatefulset(t *testing.T) {
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
			// TODO only do this for Vault image, not the other deployment
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
	require.NoError(t, ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.StatefulSet{}, ct.Namespace, "vault"))
	pods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "vault")
	require.NoError(t, err)
	require.Len(t, pods, 1)

	var token string
	loggingPath := "/vault/data/openbao.log"

	t.Run("transit engine api auto-unseals Vault", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*60*time.Second))
		defer cancel()

		vaultLogs, stdErr, err := ct.Kubeclient.ExecContainer(ctx, ct.Namespace, pods[0].Name, "openbao-server", []string{"cat", loggingPath})
		require.NoError(err, "stdout: %s, stderr: %s", vaultLogs, stdErr)

		coordinatorLogs, err := ct.Kubeclient.GetContainerLogs(ctx, ct.Namespace, "StatefulSet", "coordinator", "coordinator")
		require.NoError(err)

		// Verifies that Vault is automatically unsealed using the coordinator's transit engine API.
		require.True(checkUnsealingLogs(coordinatorLogs, vaultLogs, "vault_unsealing"))
	})

	// Extracts the root token from Vault logs, enables the KV secrets engine, and writes
	// a test secret to verify that Vault accepts and stores data correctly.
	// This secret will later be used to validate data persistence after a Vault restart.
	t.Run("Enable KV engine and create secret on Vault", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(60*time.Second))
		defer cancel()

		baoClientLogs, err := ct.Kubeclient.GetContainerLogs(ctx, ct.Namespace, "StatefulSet", "vault", "openbao-client")
		require.NoError(err)
		token, err = extractVaultRootToken(baoClientLogs)
		require.NoError(err, "failed to extract Vault root token from logs:\n%s", baoClientLogs)
		require.NotEmpty(token, "extracted token is empty")

		stdOut, stdErr, err := ct.Kubeclient.ExecContainer(
			ctx,
			ct.Namespace,
			pods[0].Name,
			"openbao-client",
			[]string{
				"sh",
				"-c",
				fmt.Sprintf("export VAULT_TOKEN=%s && bao secrets enable -path=mykv kv", token),
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
				fmt.Sprintf("export VAULT_TOKEN=%s && bao kv put mykv/hello foo=bar", token),
			},
		)
		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
	})

	// Reuses the previously extracted root token to manually seal the Vault instance.
	// Verifies that the seal command completes successfully.
	t.Run("Vault can be sealed manually", func(t *testing.T) {
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
				fmt.Sprintf("export VAULT_TOKEN=%s && bao operator seal", token),
			},
		)

		require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
		t.Logf("%s", stdOut)
	})

	// Restarts the Vault deployment and verifies it auto-unseals using the transit engine API.
	// Additionally confirms that the previously written secret still exists, ensuring data persistence.
	t.Run("Vault auto-unseals after restart and secrets are persistent", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(60*time.Second))
		defer cancel()

		require.NoError(ct.Kubeclient.Restart(ctx, kubeclient.StatefulSet{}, ct.Namespace, "vault"))
		require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.StatefulSet{}, ct.Namespace, "vault"))

		pods, err = ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "vault")
		require.NoError(err)
		require.Len(pods, 1)

		vaultLogs, stdErr, err := ct.Kubeclient.ExecContainer(ctx, ct.Namespace, pods[0].Name, "openbao-server", []string{"cat", loggingPath})
		require.NoError(err, "stdout: %s, stderr: %s", vaultLogs, stdErr)

		coordinatorLogs, err := ct.Kubeclient.GetContainerLogs(ctx, ct.Namespace, "StatefulSet", "coordinator", "coordinator")
		require.NoError(err)

		// Verifies that Vault is automatically unsealed using the coordinator's transit engine API.
		require.True(checkUnsealingLogs(coordinatorLogs, vaultLogs, "vault_unsealing"))

		stdOut, stdErr, err := ct.Kubeclient.ExecContainer(
			ctx,
			ct.Namespace,
			pods[0].Name,
			"openbao-client",
			[]string{
				"sh", "-c",
				fmt.Sprintf("export VAULT_TOKEN=%s && bao kv get mykv/hello", token),
			},
		)
		require.NoError(err, "stderr: %s", stdErr)
		require.Contains(stdOut, "foo")
		require.Contains(stdOut, "bar")
	})
}

// checkUnsealingLogs verifies that Vault was unsealed using the coordinator's transit API.
// It does this by matching the timestamp of Vault's latest unseal event
// with a corresponding /encrypt/<workloadSecretID> request in the coordinator logs.
func checkUnsealingLogs(coordinatorLogs, vaultLogs, workloadSecretID string) bool {
	var lastUnsealTime time.Time
	var foundUnseal bool

	// Step 1: Find the latest "vault is unsealed" timestamp
	vaultScanner := bufio.NewScanner(strings.NewReader(vaultLogs))
	for vaultScanner.Scan() {
		line := vaultScanner.Text()
		if strings.Contains(line, "vault is unsealed") {
			t, err := parseTimestamp(line)
			if err == nil {
				lastUnsealTime = t
				foundUnseal = true
			}
		}
	}

	if !foundUnseal {
		return false
	}

	// Step 2: Match against coordinator log lines for /encrypt/<workloadSecretID>
	coordinatorScanner := bufio.NewScanner(strings.NewReader(coordinatorLogs))
	for coordinatorScanner.Scan() {
		line := coordinatorScanner.Text()
		if strings.Contains(line, "/encrypt/"+workloadSecretID) {
			coordTime, err := parseTimestamp(line)
			if err != nil {
				continue
			}
			diff := lastUnsealTime.Sub(coordTime)
			if diff < 0 {
				diff = -diff
			}
			if diff <= time.Second {
				return true
			}
		}
	}

	return false
}

// parseTimestamp parses a formatted log line (internal Vault or Coordinator) and returns the timestamp of the logged event.
func parseTimestamp(line string) (time.Time, error) {
	var tsStr string
	if strings.HasPrefix(line, "time=") {
		// Coordinator format: time=2025-05-12T12:57:36.629Z ...
		parts := strings.Split(line, " ")
		tsStr = strings.TrimPrefix(parts[0], "time=")
	} else {
		// Vault format: 2025-05-12T12:57:36.341Z [INFO] ...
		parts := strings.Split(line, " ")
		tsStr = parts[0]
	}
	return time.Parse(time.RFC3339Nano, tsStr)
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

// TestCheckUnsealingLogs adds a unit test, ensuring that the
// checkUnsealingLogs function works properly for example input logs.
func TestCheckUnsealingLogs(t *testing.T) {
	coordinatorLogs := `
time=2025-05-12T12:57:36.629Z level=DEBUG msg="Authorized access to /encrypt/vault_unsealing "
time=2025-05-12T12:57:36.692Z level=DEBUG msg="Authorized access to /encrypt/vault_unsealing "
time=2025-05-12T12:57:36.698Z level=DEBUG msg="Authorized access to /encrypt/vault_unsealing "
time=2025-05-12T12:57:36.701Z level=DEBUG msg="Authorized access to /decrypt/vault_unsealing "
time=2025-05-12T12:57:36.712Z level=DEBUG msg="Authorized access to /encrypt/vault_unsealing "
`
	vaultLogs := `
2025-05-12T12:57:36.337Z [INFO]  core: restoring leases
2025-05-12T12:57:36.339Z [INFO]  core: usage gauge collection is disabled
2025-05-12T12:57:36.341Z [INFO]  core: post-unseal setup complete
2025-05-12T12:57:36.341Z [INFO]  core: vault is unsealed
2025-05-12T12:57:36.341Z [INFO]  core: unsealed with stored key
`

	require.True(t, checkUnsealingLogs(coordinatorLogs, vaultLogs, "vault_unsealing"))
}
