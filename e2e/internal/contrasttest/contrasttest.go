// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package contrasttest

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kubeapi"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	ksync "github.com/katexochen/sync/api/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// ContrastTest is the Contrast test helper struct.
type ContrastTest struct {
	// inputs, usually filled by New()
	Namespace             string
	WorkDir               string
	ImageReplacements     map[string]string
	ImageReplacementsFile string
	Platform              platforms.Platform
	NamespaceFile         string
	SkipUndeploy          bool
	Kubeclient            *kubeclient.Kubeclient

	// outputs of contrast subcommands
	meshCACertPEM []byte
	rootCACertPEM []byte
}

// New creates a new contrasttest.T object bound to the given test.
func New(t *testing.T, imageReplacements, namespaceFile string, platform platforms.Platform, skipUndeploy bool) *ContrastTest {
	return &ContrastTest{
		Namespace:             MakeNamespace(t),
		WorkDir:               t.TempDir(),
		ImageReplacementsFile: imageReplacements,
		Platform:              platform,
		NamespaceFile:         namespaceFile,
		SkipUndeploy:          skipUndeploy,
		Kubeclient:            kubeclient.NewForTest(t),
	}
}

// Init patches the given resources for the test environment and makes them available to Generate and Set.
func (ct *ContrastTest) Init(t *testing.T, resources []any) {
	require := require.New(t)

	f, err := os.Open(ct.ImageReplacementsFile)
	require.NoError(err, "Image replacements %s file not found", ct.ImageReplacementsFile)
	ct.ImageReplacements, err = kuberesource.ImageReplacementsFromFile(f)
	require.NoError(err, "Parsing image replacements from %s failed", ct.ImageReplacementsFile)

	// If available, acquire a fifo ticket to synchronize cluster access with
	// other running e2e tests. We request a ticket and wait for our turn.
	// Ticket is released in the cleanup function. The sync server will ensure
	// that only one test is using the cluster at a time.
	var fifo *ksync.Fifo
	if fifoUUID, ok := os.LookupEnv("SYNC_FIFO_UUID"); ok {
		syncEndpoint, ok := os.LookupEnv("SYNC_ENDPOINT")
		require.True(ok, "SYNC_ENDPOINT must be set when SYNC_FIFO_UUID is set")
		t.Logf("Syncing with fifo %s of endpoint %s", fifoUUID, syncEndpoint)
		fifo = ksync.FifoFromUUID(syncEndpoint, fifoUUID)
		err := fifo.TicketAndWait(context.Background())
		if err != nil {
			t.Log("If this throws a 404, likely the sync server was restarted.")
			t.Log("Run 'nix run .#scripts.renew-sync-fifo' against the CI cluster to fix it.")
			require.NoError(err)
		}
		t.Logf("Acquired lock on fifo %s", fifoUUID)
	}

	// Create namespace
	namespace, err := kuberesource.ResourcesToUnstructured([]any{kuberesource.Namespace(ct.Namespace)})
	require.NoError(err)
	if ct.NamespaceFile != "" {
		require.NoError(os.WriteFile(ct.NamespaceFile, []byte(ct.Namespace), 0o644))
	}
	// Creating a namespace should not take too long.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = ct.Kubeclient.Apply(ctx, namespace...)
	cancel()
	require.NoError(err)

	t.Cleanup(func() {
		// Deleting the namespace may take some time due to pod cleanup, but we don't want to wait until the test times out.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if t.Failed() {
			ct.Kubeclient.LogDebugInfo(ctx)
		}

		if !ct.SkipUndeploy {
			// Deleting the namespace sometimes fails when the cluster is
			// unavailable (e.g. after a K3s restart). Retry deleting for up to
			// 30 seconds.
			for range 30 {
				if err := ct.Kubeclient.Delete(ctx, namespace...); err != nil {
					t.Logf("Could not delete namespace %q: %v", ct.Namespace, err)
					time.Sleep(1 * time.Second)
				} else {
					break
				}
			}
		}

		if fifo != nil {
			if err := fifo.Done(ctx); err != nil {
				t.Logf("Could not mark fifo ticket as done: %v", err)
			}
		}
	})

	// Prepare resources
	resources = kuberesource.PatchImages(resources, ct.ImageReplacements)
	resources = kuberesource.PatchNamespaces(resources, ct.Namespace)
	resources = kuberesource.PatchServiceMeshAdminInterface(resources, 9901)
	resources = kuberesource.PatchCoordinatorMetrics(resources, 9102)
	resources = kuberesource.AddLogging(resources, "debug", "*")
	unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)

	// Write resources to this test's tempdir.
	buf, err := kuberesource.EncodeUnstructured(unstructuredResources)
	require.NoError(err)
	require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yml"), buf, 0o644))

	ct.installRuntime(t)
}

// Generate runs the contrast generate command.
func (ct *ContrastTest) Generate(t *testing.T) {
	require := require.New(t)

	args := append(
		ct.commonArgs(),
		"--image-replacements", ct.ImageReplacementsFile,
		"--reference-values", ct.Platform.String(),
		path.Join(ct.WorkDir, "resources.yml"),
	)

	generate := cmd.NewGenerateCmd()
	generate.Flags().String("workspace-dir", "", "") // Make generate aware of root flags
	generate.Flags().String("log-level", "debug", "")
	generate.SetArgs(args)
	generate.SetOut(io.Discard)
	errBuf := &bytes.Buffer{}
	generate.SetErr(errBuf)

	require.NoError(generate.Execute(), "could not generate manifest: %s", errBuf)
	hash, err := os.ReadFile(path.Join(ct.WorkDir, "coordinator-policy.sha256"))
	require.NoError(err)
	require.NotEmpty(hash, "expected apply to fill coordinator policy hash")

	ct.patchReferenceValues(t, ct.Platform)
}

// patchReferenceValues modifies the manifest to contain multiple reference values for testing
// cases with multiple validators, as well as filling in bare-metal SNP-specific values.
func (ct *ContrastTest) patchReferenceValues(t *testing.T, platform platforms.Platform) {
	manifestBytes, err := os.ReadFile(ct.WorkDir + "/manifest.json")
	require.NoError(t, err)
	var m manifest.Manifest
	require.NoError(t, json.Unmarshal(manifestBytes, &m))

	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		// Duplicate the reference values to test multiple validators by having at least 2.
		m.ReferenceValues.SNP = append(m.ReferenceValues.SNP, m.ReferenceValues.SNP[len(m.ReferenceValues.SNP)-1])

		// Make the last set of reference values invalid by changing the SVNs.
		m.ReferenceValues.SNP[len(m.ReferenceValues.SNP)-1].MinimumTCB = manifest.SNPTCB{
			BootloaderVersion: toPtr(manifest.SVN(255)),
			TEEVersion:        toPtr(manifest.SVN(255)),
			SNPVersion:        toPtr(manifest.SVN(255)),
			MicrocodeVersion:  toPtr(manifest.SVN(255)),
		}
	case platforms.K3sQEMUSNP:
		// The generate command doesn't fill in all required fields when
		// generating a manifest for baremetal SNP. Do that now.
		for i, snp := range m.ReferenceValues.SNP {
			snp.MinimumTCB.BootloaderVersion = toPtr(manifest.SVN(0))
			snp.MinimumTCB.TEEVersion = toPtr(manifest.SVN(0))
			snp.MinimumTCB.SNPVersion = toPtr(manifest.SVN(0))
			snp.MinimumTCB.MicrocodeVersion = toPtr(manifest.SVN(0))
			m.ReferenceValues.SNP[i] = snp
		}
	case platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
		// The generate command doesn't fill in all required fields when
		// generating a manifest for baremetal TDX. Do that now.
		for i, tdx := range m.ReferenceValues.TDX {
			tdx.MinimumTeeTcbSvn = manifest.HexString("04010200000000000000000000000000")
			tdx.MrSeam = manifest.HexString("1cc6a17ab799e9a693fac7536be61c12ee1e0fabada82d0c999e08ccee2aa86de77b0870f558c570e7ffe55d6d47fa04")
			m.ReferenceValues.TDX[i] = tdx
		}
	}

	manifestBytes, err = json.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(ct.WorkDir+"/manifest.json", manifestBytes, 0o644))
}

// Apply the generated resources to the Kubernetes test environment.
func (ct *ContrastTest) Apply(t *testing.T) {
	require := require.New(t)

	yaml, err := os.ReadFile(path.Join(ct.WorkDir, "resources.yml"))
	require.NoError(err)
	objects, err := kubeapi.UnmarshalUnstructuredK8SResource(yaml)
	require.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	require.NoError(ct.Kubeclient.Apply(ctx, objects...))
}

// Set runs the contrast set subcommand.
func (ct *ContrastTest) Set(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	require.NoError(ct.runAgainstCoordinator(ctx, cmd.NewSetCmd(), path.Join(ct.WorkDir, "resources.yml")))
}

// RunVerify runs the contrast verify subcommand.
func (ct *ContrastTest) RunVerify() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := ct.runAgainstCoordinator(ctx, cmd.NewVerifyCmd()); err != nil {
		return err
	}

	var err error
	ct.meshCACertPEM, err = os.ReadFile(path.Join(ct.WorkDir, "mesh-ca.pem"))
	if err != nil {
		return fmt.Errorf("no mesh ca cert: %w", err)
	}
	ct.rootCACertPEM, err = os.ReadFile(path.Join(ct.WorkDir, "coordinator-root-ca.pem"))
	if err != nil {
		return fmt.Errorf("no root ca cert: %w", err)
	}
	return nil
}

// Verify runs the contrast verify subcommand and fails the test if it is not successful.
func (ct *ContrastTest) Verify(t *testing.T) {
	require.NoError(t, ct.RunVerify())
}

// Recover runs the contrast recover subcommand.
func (ct *ContrastTest) Recover(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	require.NoError(ct.runAgainstCoordinator(ctx, cmd.NewRecoverCmd()))
}

// MeshCACert returns a CertPool that contains the coordinator mesh CA cert.
func (ct *ContrastTest) MeshCACert() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ct.meshCACertPEM)
	return pool
}

// RootCACert returns a CertPool that contains the coordinator root CA cert.
func (ct *ContrastTest) RootCACert() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ct.rootCACertPEM)
	return pool
}

func (ct *ContrastTest) commonArgs() []string {
	return []string{
		"--workspace-dir", ct.WorkDir,
	}
}

// installRuntime initializes the kubernetes runtime class for the test.
func (ct *ContrastTest) installRuntime(t *testing.T) {
	require := require.New(t)

	resources, err := kuberesource.Runtime(ct.Platform)
	require.NoError(err)
	resources = kuberesource.PatchImages(resources, ct.ImageReplacements)
	resources = kuberesource.PatchNamespaces(resources, ct.Namespace)

	unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	require.NoError(ct.Kubeclient.Apply(ctx, unstructuredResources...))

	require.NoError(ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.DaemonSet{}, ct.Namespace, "contrast-node-installer"))
}

// runAgainstCoordinator forwards the coordinator port and executes the command against it.
func (ct *ContrastTest) runAgainstCoordinator(ctx context.Context, cmd *cobra.Command, args ...string) error {
	policyHash, err := os.ReadFile(path.Join(ct.WorkDir, "coordinator-policy.sha256"))
	if err != nil {
		return fmt.Errorf("reading coordinator policy hash: %w", err)
	}
	if len(policyHash) == 0 {
		return fmt.Errorf("coordinator policy hash cannot be empty")
	}

	if err := ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.StatefulSet{}, ct.Namespace, "coordinator"); err != nil {
		return fmt.Errorf("waiting for coordinator: %w", err)
	}
	if err := ct.Kubeclient.WaitFor(ctx, kubeclient.Ready, kubeclient.Pod{}, ct.Namespace, "port-forwarder-coordinator"); err != nil {
		return fmt.Errorf("waiting for port-forwarder-coordinator: %w", err)
	}

	// Make the subcommand aware of the persistent flag.
	// Do it outside the closure because declaring a flag twice panics.
	cmd.Flags().String("workspace-dir", "", "")

	return ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-coordinator", "1313", func(addr string) error {
		commonArgs := append(ct.commonArgs(),
			"--coordinator-policy-hash", string(policyHash),
			"--coordinator", addr)
		cmd.SetArgs(append(commonArgs, args...))
		cmd.SetOut(io.Discard)
		errBuf := &bytes.Buffer{}
		cmd.SetErr(errBuf)

		if err := cmd.Execute(); err != nil {
			return fmt.Errorf("running %q: %s", cmd.Use, errBuf)
		}
		return nil
	})
}

// FactorPlatformTimeout returns a timeout that is adjusted for the platform.
// Baseline is AKS.
func (ct *ContrastTest) FactorPlatformTimeout(timeout time.Duration) time.Duration {
	switch ct.Platform {
	case platforms.AKSCloudHypervisorSNP: // AKS defined is the baseline
		return timeout
	case platforms.K3sQEMUSNP, platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
		return 2 * timeout
	default:
		return timeout
	}
}

// MakeNamespace creates a namespace string using a given *testing.T.
func MakeNamespace(t *testing.T) string {
	buf := make([]byte, 4)
	re := regexp.MustCompile("[a-z0-9-]+")
	n, err := rand.Reader.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)

	return strings.Join(append(re.FindAllString(strings.ToLower(t.Name()), -1), hex.EncodeToString(buf)), "-")
}

func toPtr[T any](t T) *T {
	return &t
}
