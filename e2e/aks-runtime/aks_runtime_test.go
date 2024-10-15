// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package aksruntime

import (
	"context"
	"flag"
	"os"
	"path"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/confcom"
	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kubeapi"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

const (
	opensslFrontend = "openssl-frontend"
	opensslBackend  = "openssl-backend"
)

var (
	imageReplacementsFile, namespaceFile, platformStr string
	skipUndeploy                                      bool
)

func TestAKSRuntime(t *testing.T) {
	// TODO: Log kata information
	require := require.New(t)

	workdir := t.TempDir()

	f, err := os.Open(imageReplacementsFile)
	require.NoError(err)
	imageReplacements, err := kuberesource.ImageReplacementsFromFile(f)
	require.NoError(err)
	namespace := contrasttest.MakeNamespace(t)

	c := kubeclient.NewForTest(t)

	// create the namespace
	ns, err := kuberesource.ResourcesToUnstructured([]any{kuberesource.Namespace(namespace)})
	require.NoError(err)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = c.Apply(ctx, ns...)
	cancel()
	require.NoError(err)
	if namespaceFile != "" {
		require.NoError(os.WriteFile(namespaceFile, []byte(namespace), 0o644))
	}

	// define resources
	resources := kuberesource.OpenSSL()
	// TODO: check if this can be removed since they are overwritten later
	resources = kuberesource.PatchRuntimeHandlers(resources, "kata-cc-isolation")
	resources = kuberesource.PatchNamespaces(resources, namespace)
	resources = kuberesource.PatchImages(resources, imageReplacements)

	toWrite, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)

	t.Log("generating policies...")
	// generate policies
	resourceBytes, err := kuberesource.EncodeUnstructured(toWrite)
	require.NoError(err)
	require.NoError(os.WriteFile(path.Join(workdir, "resources.yaml"), resourceBytes, 0o644))
	require.NoError(confcom.KataPolicyGen(t, path.Join(workdir, "resources.yaml")))

	//
	// platform, err := platforms.FromString(platformStr)
	// require.NoError(err)
	// args := []string{
	// 	"--image-replacements", imageReplacementsFile,
	// 	"--reference-values", platform.String(),
	// 	path.Join(workdir, "resources.yaml"),
	// }
	//
	// generate := cmd.NewGenerateCmd()
	// generate.Flags().String("workspace-dir", "", "") // Make generate aware of root flags
	// generate.Flags().String("log-level", "debug", "")
	// generate.SetArgs(args)
	// generate.SetOut(io.Discard)
	// errBuf := &bytes.Buffer{}
	// generate.SetErr(errBuf)

	// load in generated resources and patch the runtime handler again
	resourceBytes, err = os.ReadFile(path.Join(workdir, "resources.yaml"))
	require.NoError(err)
	toApply, err := kubeapi.UnmarshalUnstructuredK8SResource(resourceBytes)
	require.NoError(err)
	t.Logf("%#v", toApply)

	t.Log("policies generated!")

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	err = c.Apply(ctx, toApply...)
	require.NoError(err)
	require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, namespace, opensslBackend))
	require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, namespace, opensslFrontend))
	c.LogDebugInfo(context.Background())
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.StringVar(&platformStr, "platform", "", "Deployment platform")
	flag.BoolVar(&skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}
