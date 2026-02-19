// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package contrasttest

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/cryptohelpers"
	"github.com/edgelesssys/contrast/internal/httpapi"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/edgelesssys/contrast/sdk"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Flags contains the parsed Flags for the test.
var Flags testFlags

// testFlags contains the flags for the test.
type testFlags struct {
	PlatformStr                 string
	ImageReplacementsFile       string
	NamespaceFile               string
	NamespaceSuffix             string
	NodeInstallerTargetConfType string
	SyncTicketFile              string
	InsecureEnableDebugShell    bool
}

// RegisterFlags registers the flags that are shared between all tests.
func RegisterFlags() {
	flag.StringVar(&Flags.ImageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&Flags.NamespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.StringVar(&Flags.NamespaceSuffix, "namespace-suffix", "", "suffix to append to the namespace")
	flag.StringVar(&Flags.PlatformStr, "platform", "", "Deployment platform")
	flag.StringVar(&Flags.NodeInstallerTargetConfType, "node-installer-target-conf-type", "", "Type of node installer target configuration to generate (k3s,...)")
	flag.StringVar(&Flags.SyncTicketFile, "sync-ticket-file", "", "file that contains the sync ticket")
	flag.BoolVar(&Flags.InsecureEnableDebugShell, "insecure-enable-debug-shell-access", false, "enable the debug shell service")
}

// ContrastTest is the Contrast test helper struct.
type ContrastTest struct {
	// inputs, usually filled by New()
	Namespace                      string
	WorkDir                        string
	ImageReplacements              map[string]string
	ImageReplacementsFile          string
	Platform                       platforms.Platform
	NamespaceFile                  string
	RuntimeClassName               string
	NodeInstallerTargetConfType    string
	NodeInstallerImagePullerConfig []byte
	GHCRToken                      string
	Kubeclient                     *kubeclient.Kubeclient

	// outputs of contrast subcommands
	meshCACertPEM []byte
	rootCACertPEM []byte
}

// New creates a new contrasttest.T object bound to the given test.
func New(t *testing.T) *ContrastTest {
	require := require.New(t)

	platform, err := platforms.FromString(Flags.PlatformStr)
	require.NoError(err)

	runtimeClass, err := kuberesource.ContrastRuntimeClass(platform)
	require.NoError(err)

	workDir := t.TempDir()
	t.Setenv(constants.CacheDirEnvVar, workDir)

	ct := &ContrastTest{
		Namespace:                   MakeNamespace(t, Flags.NamespaceSuffix),
		WorkDir:                     workDir,
		ImageReplacementsFile:       Flags.ImageReplacementsFile,
		Platform:                    platform,
		NamespaceFile:               Flags.NamespaceFile,
		RuntimeClassName:            *runtimeClass.Handler,
		Kubeclient:                  kubeclient.NewForTest(t),
		NodeInstallerTargetConfType: Flags.NodeInstallerTargetConfType,
	}

	token := os.Getenv("CONTRAST_GHCR_READ")
	if token != "" {
		cfg := map[string]any{
			"registries": map[string]any{
				"ghcr.io.": map[string]string{
					"auth": base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "user-not-required-here:%s", token)),
				},
			},
		}
		imagePullerConfig, err := toml.Marshal(cfg)
		require.NoError(err)

		ct.GHCRToken = token
		ct.NodeInstallerImagePullerConfig = imagePullerConfig
	}

	return ct
}

// Init patches the given resources for the test environment and makes them available to Generate and Set.
func (ct *ContrastTest) Init(t *testing.T, resources []any) {
	require := require.New(t)

	f, err := os.Open(ct.ImageReplacementsFile)
	require.NoError(err, "Image replacements %s file not found", ct.ImageReplacementsFile)
	ct.ImageReplacements, err = kuberesource.ImageReplacementsFromFile(f)
	require.NoError(err, "Parsing image replacements from %s failed", ct.ImageReplacementsFile)

	// Create namespace
	namespace := kuberesource.Namespace(ct.Namespace)

	// Add sync ticket label if provided
	if ticket, err := os.ReadFile(Flags.SyncTicketFile); err == nil {
		namespace.WithLabels(map[string]string{
			"contrast.edgeless.systems/sync-ticket": strings.TrimSpace(string(ticket)),
		})
	} else {
		require.ErrorIs(err, os.ErrNotExist, "Reading sync ticket from %s failed", Flags.SyncTicketFile)
	}

	namespaceUnstr, err := kuberesource.ResourcesToUnstructured([]any{namespace})
	require.NoError(err)
	if ct.NamespaceFile != "" {
		file, err := os.OpenFile(ct.NamespaceFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
		require.NoError(err)
		defer file.Close()
		_, err = file.WriteString(ct.Namespace + "\n")
		require.NoError(err)
	}
	// Creating a namespace should not take too long.
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	err = ct.Kubeclient.Apply(ctx, namespaceUnstr...)
	cancel()
	require.NoError(err)

	t.Cleanup(func() {
		// Deleting the namespace may take some time due to pod cleanup, but we don't want to wait until the test times out.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if t.Failed() {
			ct.Kubeclient.LogDebugInfo(ctx)
		}
	})

	// Prepare resources
	resources = kuberesource.PatchImages(resources, ct.ImageReplacements)
	resources = kuberesource.PatchDockerSecrets(resources, ct.Namespace, ct.GHCRToken)
	resources = kuberesource.PatchNamespaces(resources, ct.Namespace)
	resources = kuberesource.PatchCoordinatorMetrics(resources)
	resources = kuberesource.AddLogging(resources, "debug", "*")
	unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)

	// Write resources to this test's tempdir.
	buf, err := kuberesource.EncodeUnstructured(unstructuredResources)
	require.NoError(err)
	require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yml"), buf, 0o644))

	ct.installRuntime(t, resources)
}

// Generate runs the contrast generate command and fails the test if the command fails.
func (ct *ContrastTest) Generate(t *testing.T) {
	require.NoError(t, ct.RunGenerate(t.Context()))
}

// RunGenerate runs the contrast generate command.
func (ct *ContrastTest) RunGenerate(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	args := append(
		ct.commonArgs(),
		"--image-replacements", ct.ImageReplacementsFile,
		"--reference-values", ct.Platform.String(),
		fmt.Sprintf("--insecure-enable-debug-shell-access=%t", Flags.InsecureEnableDebugShell),
		ct.WorkDir,
	)

	generate := cmd.NewGenerateCmd()
	generate.Flags().String("workspace-dir", "", "") // Make generate aware of root flags
	generate.Flags().String("log-level", "debug", "")
	generate.SetArgs(args)
	generate.SetOut(io.Discard)
	errBuf := &bytes.Buffer{}
	generate.SetErr(errBuf)

	if err := generate.ExecuteContext(ctx); err != nil {
		return errors.Join(fmt.Errorf("%s", errBuf), err)
	}
	patchRefValsFunc, err := PatchReferenceValues(ctx, ct.Kubeclient)
	if err != nil {
		return fmt.Errorf("getting func to patch reference values in manifest: %w", err)
	}
	if err := ct.RunPatchManifest(patchRefValsFunc); err != nil {
		return fmt.Errorf("patching manifest with reference values: %w", err)
	}
	return nil
}

// ManifestPath returns the full path to the manifest file used by this instance.
func (ct *ContrastTest) ManifestPath() string {
	return filepath.Join(ct.WorkDir, "manifest.json")
}

// PatchManifestFunc defines a function type allowing the given manifest to be modified.
type PatchManifestFunc func(manifest.Manifest) (manifest.Manifest, error)

// PatchManifest modifies the current manifest by executing a provided PatchManifestFunc on it. This function fails the test if it encounters an error.
func (ct *ContrastTest) PatchManifest(t *testing.T, patchFn PatchManifestFunc) {
	require.NoError(t, ct.RunPatchManifest(patchFn))
}

// RunPatchManifest modifies the current manifest by executing a provided PatchManifestFunc on it.
func (ct *ContrastTest) RunPatchManifest(patchFn PatchManifestFunc) error {
	manifestBytes, err := os.ReadFile(ct.ManifestPath())
	if err != nil {
		return err
	}
	var m manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return err
	}
	patchedManifest, err := patchFn(m)
	if err != nil {
		return fmt.Errorf("patching manifest: %w", err)
	}
	manifestBytes, err = json.Marshal(patchedManifest)
	if err != nil {
		return err
	}
	if err := os.WriteFile(ct.ManifestPath(), manifestBytes, 0o644); err != nil {
		return err
	}
	return nil
}

// PatchReferenceValues returns a PatchManifestFunc which modifies the reference values in a manifest
// based on the 'bm-tcb-specs' ConfigMap persistently stored in the 'default' namespace.
func PatchReferenceValues(ctx context.Context, k *kubeclient.Kubeclient) (PatchManifestFunc, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	configMap, err := k.Client.CoreV1().ConfigMaps("default").Get(ctx, "bm-tcb-specs", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting ConfigMap bm-tcb-specs: %w", err)
	}
	var patches manifest.ReferenceValuePatches
	if err := json.Unmarshal([]byte(configMap.Data["specs"]), &patches); err != nil {
		return nil, fmt.Errorf("unmarshaling patches: %w", err)
	}
	return func(m manifest.Manifest) (manifest.Manifest, error) {
		if err := m.ReferenceValues.Patch(patches); err != nil {
			return m, err
		}
		m.ReferenceValues.PurgeEmpty()
		return m, nil
	}, nil
}

// Apply the generated resources to the Kubernetes test environment.
func (ct *ContrastTest) Apply(t *testing.T) {
	require := require.New(t)

	ymlFiles, err := fs.Glob(os.DirFS(ct.WorkDir), "*.yml")
	require.NoError(err)
	yamlFiles, err := fs.Glob(os.DirFS(ct.WorkDir), "*.yaml")
	require.NoError(err)
	yamlFiles = append(yamlFiles, ymlFiles...)
	var files []string
	for _, file := range yamlFiles {
		files = append(files, path.Join(ct.WorkDir, file))
	}

	require.NoError(err)
	for _, file := range files {
		yaml, err := os.ReadFile(file)
		require.NoError(err)
		ct.ApplyFromYAML(t, yaml)
	}
}

// ApplyFromYAML applies the given YAML to the Kubernetes test environment.
func (ct *ContrastTest) ApplyFromYAML(t *testing.T, yaml []byte) {
	require := require.New(t)

	objects, err := kuberesource.UnmarshalUnstructuredK8SResource(yaml)
	require.NoError(err)

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Minute)
	defer cancel()

	require.NoError(ct.Kubeclient.Apply(ctx, objects...))
}

// RunSet runs the contrast set subcommand.
func (ct *ContrastTest) RunSet(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	return ct.runAgainstCoordinator(ctx, cmd.NewSetCmd(), ct.WorkDir)
}

// Set runs the contrast set subcommand and fails the test if it is not successful.
func (ct *ContrastTest) Set(t *testing.T) {
	require.NoError(t, ct.RunSet(t.Context()))
}

// RunVerify runs the contrast verify subcommand.
func (ct *ContrastTest) RunVerify(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	if err := ct.runAgainstCoordinator(ctx, cmd.NewVerifyCmd()); err != nil {
		return err
	}

	var err error
	ct.meshCACertPEM, err = os.ReadFile(path.Join(ct.WorkDir, "verify", "mesh-ca.pem"))
	if err != nil {
		return fmt.Errorf("no mesh ca cert: %w", err)
	}
	ct.rootCACertPEM, err = os.ReadFile(path.Join(ct.WorkDir, "verify", "coordinator-root-ca.pem"))
	if err != nil {
		return fmt.Errorf("no root ca cert: %w", err)
	}

	// Test the HTTP API attestation endpoint, too.

	cacheDir := os.Getenv(constants.CacheDirEnvVar)
	if cacheDir == "" {
		return fmt.Errorf("contrasttest.New should have set env var %q, but it's empty", constants.CacheDirEnvVar)
	}
	client := sdk.New().WithFSStore(afero.NewBasePathFs(afero.NewOsFs(), cacheDir))
	nonce, err := cryptohelpers.GenerateRandomBytes(cryptohelpers.RNGLengthDefault)
	if err != nil {
		return fmt.Errorf("generating nonce: %w", err)
	}
	var serializedAttestation []byte
	err = ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-coordinator", httpapi.Port, func(addr string) error {
		url := fmt.Sprintf("http://%s/attest", addr)
		resp, err := client.GetAttestation(ctx, url, nonce)
		if err != nil {
			return fmt.Errorf("getting attestation: %w", err)
		}
		serializedAttestation = resp
		return nil
	})
	if err != nil {
		return fmt.Errorf("calling HTTP API: %w", err)
	}

	state, err := client.ValidateAttestation(ctx, nonce, serializedAttestation)
	if err != nil {
		return fmt.Errorf("validating attestation: %w", err)
	}

	expectedManifest, err := os.ReadFile(ct.ManifestPath())
	if err != nil {
		return fmt.Errorf("reading manifest from workspace: %w", err)
	}
	manifest := state.Manifests[len(state.Manifests)-1]
	if !bytes.Equal(expectedManifest, manifest) {
		return fmt.Errorf("manifests don't match.\nExpected:\n%s\nActual:\n%s", string(expectedManifest), string(manifest))
	}
	if !bytes.Equal(ct.rootCACertPEM, state.RootCA) {
		return fmt.Errorf("root CA certs don't match.\nExpected:\n%s\nActual:\n%s", string(ct.rootCACertPEM), string(state.RootCA))
	}
	if !bytes.Equal(ct.meshCACertPEM, state.MeshCA) {
		return fmt.Errorf("mesh CA certs don't match.\nExpected:\n%s\nActual:\n%s", string(ct.meshCACertPEM), string(state.MeshCA))
	}

	return nil
}

// Verify runs the contrast verify subcommand and fails the test if it is not successful.
func (ct *ContrastTest) Verify(t *testing.T) {
	require.NoError(t, ct.RunVerify(t.Context()))
}

// Recover runs the contrast recover subcommand and fails the test if it is not successful.
func (ct *ContrastTest) Recover(t *testing.T) {
	require.NoError(t, ct.runAgainstCoordinator(t.Context(), cmd.NewRecoverCmd()))
}

// RunRecover runs the contrast recover subcommand.
func (ct *ContrastTest) RunRecover(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	return ct.runAgainstCoordinator(ctx, cmd.NewRecoverCmd())
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
func (ct *ContrastTest) installRuntime(t *testing.T, resources []any) {
	require := require.New(t)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	var nodeInstallerDeps []any
	if ct.NodeInstallerTargetConfType != "" && ct.NodeInstallerTargetConfType != "none" {
		nodeInstallerTargetConf, err := kuberesource.NodeInstallerTargetConfig(ct.NodeInstallerTargetConfType)
		require.NoError(err)
		nodeInstallerDeps = append(nodeInstallerDeps, nodeInstallerTargetConf)
	}

	if ct.NodeInstallerImagePullerConfig != nil {
		imagePullSecret := kuberesource.NodeInstallerImagePullerSecret(ct.Namespace, ct.NodeInstallerImagePullerConfig)
		nodeInstallerDeps = append(nodeInstallerDeps, imagePullSecret)
	}

	if len(nodeInstallerDeps) > 0 {
		nodeInstallerDeps = kuberesource.PatchNamespaces(nodeInstallerDeps, ct.Namespace)
		unstructured, err := kuberesource.ResourcesToUnstructured(nodeInstallerDeps)
		require.NoError(err)
		require.NoError(ct.Kubeclient.Apply(ctx, unstructured...))
	}

	resources, err := kuberesource.Runtimes(ct.Platform, resources)
	require.NoError(err)
	resources, err = kuberesource.PatchNodeInstallers(resources, ct.Platform)
	require.NoError(err)
	resources = kuberesource.PatchImages(resources, ct.ImageReplacements)
	resources = kuberesource.PatchNamespaces(resources, ct.Namespace)

	unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)

	require.NoError(ct.Kubeclient.Apply(ctx, unstructuredResources...))

	for _, r := range unstructuredResources {
		if r.GetKind() != "DaemonSet" {
			continue
		}

		require.NoError(ct.Kubeclient.WaitForDaemonSet(ctx, ct.Namespace, r.GetName()))
	}
}

// runAgainstCoordinator forwards the coordinator port and executes the command against it.
func (ct *ContrastTest) runAgainstCoordinator(ctx context.Context, cmd *cobra.Command, args ...string) error {
	if err := ct.Kubeclient.WaitForCoordinator(ctx, ct.Namespace); err != nil {
		return fmt.Errorf("waiting for coordinator: %w", err)
	}

	if err := ct.Kubeclient.WaitForPod(ctx, ct.Namespace, "port-forwarder-coordinator"); err != nil {
		return fmt.Errorf("waiting for port-forwarder-coordinator: %w", err)
	}

	// Make the subcommand aware of the persistent flags.
	// Do it outside the closure because declaring a flag twice panics.
	cmd.Flags().String("workspace-dir", "", "")
	cmd.Flags().String("log-level", "debug", "")

	return ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-coordinator", userapi.Port, func(addr string) error {
		// Go never uses a proxy for connections to localhost. To enable proxy tests, we
		// replace localhost with 0.0.0.0, which can be used as localhost on Linux and BSD.
		addr = strings.Replace(addr, "localhost", "0.0.0.0", 1)

		commonArgs := append(ct.commonArgs(), "--coordinator", addr)
		cmd.SetArgs(append(commonArgs, args...))
		cmd.SetOut(io.Discard)
		errBuf := &bytes.Buffer{}
		cmd.SetErr(errBuf)

		if err := cmd.ExecuteContext(ctx); err != nil {
			return fmt.Errorf("running %q: %s", cmd.Use, errBuf)
		}
		return nil
	})
}

// FactorPlatformTimeout returns a timeout that is adjusted for the platform.
// Baseline is AKS.
func (ct *ContrastTest) FactorPlatformTimeout(timeout time.Duration) time.Duration {
	switch ct.Platform {
	case platforms.MetalQEMUSNP, platforms.MetalQEMUTDX, platforms.MetalQEMUSNPGPU, platforms.MetalQEMUTDXGPU:
		return 2 * timeout
	default:
		panic(fmt.Sprintf("FactorPlatformTimeout not configured for platform %q", ct.Platform))
	}
}

// MakeNamespace creates a namespace string using a given *testing.T.
func MakeNamespace(t *testing.T, namespaceSuffix string) string {
	var namespaceParts []string

	// First part(s) are consist of all valid characters in the lower case test name.
	re := regexp.MustCompile("[a-z0-9-]+")
	namespaceParts = append(namespaceParts, re.FindAllString(strings.ToLower(t.Name()), -1)...)

	// Append some randomness
	buf := make([]byte, 4)
	n, err := rand.Reader.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)

	namespaceParts = append(namespaceParts, fmt.Sprintf("%x", buf))

	return strings.Join(namespaceParts, "-") + namespaceSuffix
}
