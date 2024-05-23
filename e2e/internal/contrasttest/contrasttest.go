// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package contrasttest

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
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
	ksync "github.com/katexochen/sync/api/client"
	"github.com/stretchr/testify/require"
)

// ContrastTest is the Contrast test helper struct.
type ContrastTest struct {
	// inputs, usually filled by New()
	Namespace         string
	WorkDir           string
	ImageReplacements map[string]string
	Kubeclient        *kubeclient.Kubeclient

	// outputs of contrast subcommands
	coordinatorPolicyHash string
	meshCACertPEM         []byte
	rootCACertPEM         []byte
}

// New creates a new contrasttest.T object bound to the given test.
func New(t *testing.T, imageReplacements map[string]string) *ContrastTest {
	return &ContrastTest{
		Namespace:         makeNamespace(t),
		WorkDir:           t.TempDir(),
		ImageReplacements: imageReplacements,
		Kubeclient:        kubeclient.NewForTest(t),
	}
}

// Init patches the given resources for the test environment and makes them available to Generate and Set.
func (ct *ContrastTest) Init(t *testing.T, resources []any) {
	require := require.New(t)

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
		require.NoError(fifo.TicketAndWait(context.Background()))
		t.Logf("Acquired lock on fifo %s", fifoUUID)
	}

	// Create namespace
	namespace, err := kuberesource.ResourcesToUnstructured([]any{kuberesource.Namespace(ct.Namespace)})
	require.NoError(err)
	// Creating a namespace should not take too long.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = ct.Kubeclient.Apply(ctx, namespace...)
	cancel()
	require.NoError(err)

	t.Cleanup(func() {
		// Deleting the namespace may take some time due to pod cleanup, but we don't want to wait until the test times out.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := ct.Kubeclient.Delete(ctx, namespace...); err != nil {
			t.Logf("Could not delete namespace %q: %v", ct.Namespace, err)
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
	resources = kuberesource.AddLogging(resources, "debug")
	unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)

	// Write resources to this test's tempdir.
	buf, err := kuberesource.EncodeUnstructured(unstructuredResources)
	require.NoError(err)
	require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yaml"), buf, 0o644))

	ct.installRuntime(t)
}

// Generate runs the contrast generate command.
func (ct *ContrastTest) Generate(t *testing.T) {
	require := require.New(t)

	args := append(ct.commonArgs(), path.Join(ct.WorkDir, "resources.yaml"))

	generate := cmd.NewGenerateCmd()
	generate.Flags().String("workspace-dir", "", "") // Make generate aware of root flags
	generate.SetArgs(args)
	generate.SetOut(io.Discard)
	errBuf := &bytes.Buffer{}
	generate.SetErr(errBuf)

	require.NoError(generate.Execute(), "could not generate manifest: %s", errBuf)
	hash, err := os.ReadFile(path.Join(ct.WorkDir, "coordinator-policy.sha256"))
	require.NoError(err)
	require.NotEmpty(hash, "expected apply to fill coordinator policy hash")
	ct.coordinatorPolicyHash = string(hash)
}

// Apply the generated resources to the Kubernetes test environment.
func (ct *ContrastTest) Apply(t *testing.T) {
	require := require.New(t)

	yaml, err := os.ReadFile(path.Join(ct.WorkDir, "resources.yaml"))
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

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	require.NoError(ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, "coordinator"))

	coordinator, cancelPortForward, err := ct.Kubeclient.PortForwardPod(ctx, ct.Namespace, "port-forwarder-coordinator", "1313")
	require.NoError(err)
	defer cancelPortForward()

	args := append(ct.commonArgs(),
		"--coordinator-policy-hash", ct.coordinatorPolicyHash,
		"--coordinator", coordinator,
		path.Join(ct.WorkDir, "resources.yaml"))

	set := cmd.NewSetCmd()
	set.Flags().String("workspace-dir", "", "") // Make set aware of root flags
	set.SetArgs(args)
	set.SetOut(io.Discard)
	errBuf := &bytes.Buffer{}
	set.SetErr(errBuf)

	require.NoError(set.Execute(), "could not set manifest at coordinator: %s", errBuf)
}

// Verify runs the contrast verify subcommand.
func (ct *ContrastTest) Verify(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	require.NoError(ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, "coordinator"))

	coordinator, cancelPortForward, err := ct.Kubeclient.PortForwardPod(ctx, ct.Namespace, "port-forwarder-coordinator", "1313")
	require.NoError(err)
	defer cancelPortForward()

	verify := cmd.NewVerifyCmd()
	verify.SetArgs(append(
		ct.commonArgs(),
		"--coordinator-policy-hash", ct.coordinatorPolicyHash,
		"--coordinator", coordinator,
	))
	verify.SetOut(io.Discard)
	errBuf := &bytes.Buffer{}
	verify.SetErr(errBuf)

	require.NoError(verify.Execute(), "could not verify coordinator: %s", errBuf)

	ct.meshCACertPEM, err = os.ReadFile(path.Join(ct.WorkDir, "mesh-ca.pem"))
	require.NoError(err)
	ct.rootCACertPEM, err = os.ReadFile(path.Join(ct.WorkDir, "coordinator-root-ca.pem"))
	require.NoError(err)
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

	resources := kuberesource.Runtime()
	resources = kuberesource.PatchImages(resources, ct.ImageReplacements)
	resources = kuberesource.PatchNamespaces(resources, ct.Namespace)

	unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	require.NoError(ct.Kubeclient.Apply(ctx, unstructuredResources...))

	require.NoError(ct.Kubeclient.WaitForDaemonset(ctx, ct.Namespace, "contrast-node-installer"))
}

func makeNamespace(t *testing.T) string {
	buf := make([]byte, 4)
	re := regexp.MustCompile("[a-z0-9-]+")
	n, err := rand.Reader.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)

	return strings.Join(append(re.FindAllString(strings.ToLower(t.Name()), -1), hex.EncodeToString(buf)), "-")
}
