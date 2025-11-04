// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package contrasttest

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
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
}

// RegisterFlags registers the flags that are shared between all tests.
func RegisterFlags() {
	flag.StringVar(&Flags.ImageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&Flags.NamespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.StringVar(&Flags.NamespaceSuffix, "namespace-suffix", "", "suffix to append to the namespace")
	flag.StringVar(&Flags.PlatformStr, "platform", "", "Deployment platform")
	flag.StringVar(&Flags.NodeInstallerTargetConfType, "node-installer-target-conf-type", "", "Type of node installer target configuration to generate (k3s,...)")
	flag.StringVar(&Flags.SyncTicketFile, "sync-ticket-file", "", "file that contains the sync ticket")
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

	return &ContrastTest{
		Namespace:                   MakeNamespace(t, Flags.NamespaceSuffix),
		WorkDir:                     t.TempDir(),
		ImageReplacementsFile:       Flags.ImageReplacementsFile,
		Platform:                    platform,
		NamespaceFile:               Flags.NamespaceFile,
		RuntimeClassName:            *runtimeClass.Handler,
		Kubeclient:                  kubeclient.NewForTest(t),
		NodeInstallerTargetConfType: Flags.NodeInstallerTargetConfType,
	}
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
		require.NoError(os.WriteFile(ct.NamespaceFile, []byte(ct.Namespace), 0o644))
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
	resources = kuberesource.PatchNamespaces(resources, ct.Namespace)
	resources = kuberesource.PatchCoordinatorMetrics(resources)
	resources = kuberesource.AddLogging(resources, "debug", "*")
	unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)

	// Write resources to this test's tempdir.
	buf, err := kuberesource.EncodeUnstructured(unstructuredResources)
	require.NoError(err)
	require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yml"), buf, 0o644))

	ct.installRuntime(t)
}

// Generate runs the contrast generate command and fails the test if the command fails.
func (ct *ContrastTest) Generate(t *testing.T) {
	require.NoError(t, ct.RunGenerate(t.Context()))
}

// RunGenerate runs the contrast generate command.
func (ct *ContrastTest) RunGenerate(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	args := append(
		ct.commonArgs(),
		"--image-replacements", ct.ImageReplacementsFile,
		"--reference-values", ct.Platform.String(),
		ct.WorkDir,
	)

	generate := cmd.NewGenerateCmd()
	generate.Flags().String("workspace-dir", "", "") // Make generate aware of root flags
	generate.Flags().String("log-level", "debug", "")
	generate.SetArgs(args)
	generate.SetOut(io.Discard)
	errBuf := &bytes.Buffer{}
	generate.SetErr(errBuf)

	if err := generate.Execute(); err != nil {
		return errors.Join(fmt.Errorf("%s", errBuf), err)
	}
	patchRefValsFunc, err := PatchReferenceValues(ctx, ct.Kubeclient, ct.Platform)
	if err != nil {
		return fmt.Errorf("getting func to patch reference values in manifest: %w", err)
	}
	if err := ct.RunPatchManifest(patchRefValsFunc); err != nil {
		return fmt.Errorf("patching manifest with reference values: %w", err)
	}
	if err := ct.RunPatchManifest(addInvalidReferenceValues(ct.Platform)); err != nil {
		return fmt.Errorf("adding invalid reference values to manifest: %w", err)
	}
	return nil
}

// PatchManifestFunc defines a function type allowing the given manifest to be modified.
type PatchManifestFunc func(manifest.Manifest) manifest.Manifest

// PatchManifest modifies the current manifest by executing a provided PatchManifestFunc on it. This function fails the test if it encounters an error.
func (ct *ContrastTest) PatchManifest(t *testing.T, patchFn PatchManifestFunc) {
	require.NoError(t, ct.RunPatchManifest(patchFn))
}

// RunPatchManifest modifies the current manifest by executing a provided PatchManifestFunc on it.
func (ct *ContrastTest) RunPatchManifest(patchFn PatchManifestFunc) error {
	manifestBytes, err := os.ReadFile(ct.WorkDir + "/manifest.json")
	if err != nil {
		return err
	}
	var m manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return err
	}
	patchedManifest := patchFn(m)
	manifestBytes, err = json.Marshal(patchedManifest)
	if err != nil {
		return err
	}
	if err := os.WriteFile(ct.WorkDir+"/manifest.json", manifestBytes, 0o644); err != nil {
		return err
	}
	return nil
}

// addInvalidReferenceValues returns a PatchManifestFunc which adds a fresh, invalid entry to the specified reference values.
func addInvalidReferenceValues(platform platforms.Platform) PatchManifestFunc {
	return func(m manifest.Manifest) manifest.Manifest {
		switch platform {
		case platforms.MetalQEMUSNP, platforms.MetalQEMUSNPGPU:
			// Duplicate the reference values to test multiple validators by having at least 2.
			m.ReferenceValues.SNP = append(m.ReferenceValues.SNP, m.ReferenceValues.SNP[len(m.ReferenceValues.SNP)-1])

			// Make the last set of reference values invalid by changing the SVNs.
			m.ReferenceValues.SNP[len(m.ReferenceValues.SNP)-1].MinimumTCB = manifest.SNPTCB{
				BootloaderVersion: toPtr(manifest.SVN(255)),
				TEEVersion:        toPtr(manifest.SVN(255)),
				SNPVersion:        toPtr(manifest.SVN(255)),
				MicrocodeVersion:  toPtr(manifest.SVN(255)),
			}
		case platforms.MetalQEMUTDX:
			// Duplicate the reference values to test multiple validators by having at least 2.
			m.ReferenceValues.TDX = append(m.ReferenceValues.TDX, m.ReferenceValues.TDX[len(m.ReferenceValues.TDX)-1])

			// Make the last set of reference values invalid by changing the SVNs.
			m.ReferenceValues.TDX[len(m.ReferenceValues.TDX)-1].MrSeam = manifest.HexString("111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111")
		}
		return m
	}
}

// PatchReferenceValues returns a PatchManifestFunc which modifies the reference values in a manifest
// based on the 'bm-tcb-specs' ConfigMap persistently stored in the 'default' namespace.
func PatchReferenceValues(ctx context.Context, k *kubeclient.Kubeclient, platform platforms.Platform) (PatchManifestFunc, error) {
	var baremetalRefVal manifest.ReferenceValues
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	configMap, err := k.Client.CoreV1().ConfigMaps("default").Get(ctx, "bm-tcb-specs", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting ConfigMap bm-tcb-specs: %w", err)
	}
	err = json.Unmarshal([]byte(configMap.Data["tcb-specs.json"]), &baremetalRefVal)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling reference values: %w", err)
	}
	return func(m manifest.Manifest) manifest.Manifest {
		switch platform {
		case platforms.MetalQEMUSNP, platforms.MetalQEMUSNPGPU:
			// Overwrite the minimumTCB values with the ones loaded from the path tcbSpecificationFile.
			var snpReferenceValues []manifest.SNPReferenceValues
			for _, manifestSNP := range m.ReferenceValues.SNP {
				for _, overwriteSNP := range baremetalRefVal.SNP {
					if manifestSNP.ProductName == overwriteSNP.ProductName {
						manifestSNP.MinimumTCB = overwriteSNP.MinimumTCB
						// Filter to only use the reference values of specified baremetal SNP runners
						snpReferenceValues = append(snpReferenceValues, manifestSNP)
					}
				}
			}
			m.ReferenceValues.SNP = snpReferenceValues

		case platforms.MetalQEMUTDX:

			// Overwrite the field MrSeam with the ones loaded from the path tcbSpecificationFile.
			var tdxReferenceValues []manifest.TDXReferenceValues
			for _, manifestTDX := range m.ReferenceValues.TDX {
				for _, overwriteTDX := range baremetalRefVal.TDX {
					manifestTDX.MrSeam = overwriteTDX.MrSeam
					// Filter to only use the reference values of specified baremetal SNP runners
					tdxReferenceValues = append(tdxReferenceValues, manifestTDX)
				}
			}
			m.ReferenceValues.TDX = tdxReferenceValues

		default:
		}
		return m
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
	require.NoError(t, ct.RunVerify(t.Context()))
}

// Recover runs the contrast recover subcommand.
func (ct *ContrastTest) Recover(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
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
	if ct.NodeInstallerTargetConfType != "" && ct.NodeInstallerTargetConfType != "none" {
		nodeInstallerTargetConf, err := kuberesource.NodeInstallerTargetConfig(ct.NodeInstallerTargetConfType)
		require.NoError(err)
		resources = append(resources, nodeInstallerTargetConf)
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

	resources, err := kuberesource.Runtime(ct.Platform)
	require.NoError(err)
	resources = kuberesource.PatchImages(resources, ct.ImageReplacements)
	resources = kuberesource.PatchNamespaces(resources, ct.Namespace)

	unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

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

	return ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-coordinator", "1313", func(addr string) error {
		// Go never uses a proxy for connections to localhost. To enable proxy tests, we
		// replace localhost with 0.0.0.0, which can be used as localhost on Linux and BSD.
		addr = strings.Replace(addr, "localhost", "0.0.0.0", 1)

		commonArgs := append(ct.commonArgs(), "--coordinator", addr)
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
	case platforms.MetalQEMUSNP, platforms.MetalQEMUTDX, platforms.MetalQEMUSNPGPU:
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

func toPtr[T any](t T) *T {
	return &t
}
