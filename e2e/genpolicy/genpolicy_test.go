// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package genpolicy

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

var (
	imageReplacementsFile, namespaceFile string
	skipUndeploy                         bool
)

// TestGenpolicy runs regression tests for generated policies.
func TestGenpolicy(t *testing.T) {
	testCases := kuberesource.GenpolicyRegressionTests()

	for name, deploy := range testCases {
		t.Run(name, func(t *testing.T) {
			ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, skipUndeploy)

			ct.Init(t, []any{deploy})

			require.True(t, t.Run("generate", func(t *testing.T) {
				require := require.New(t)
				args := []string{
					"--workspace-dir", ct.WorkDir,
					"--skip-initializer",
					path.Join(ct.WorkDir, "resources.yaml"),
				}
				generate := cmd.NewGenerateCmd()
				generate.Flags().String("workspace-dir", "", "") // Make generate aware of root flags
				generate.SetArgs(args)
				generate.SetOut(io.Discard)
				errBuf := &bytes.Buffer{}
				generate.SetErr(errBuf)

				require.NoError(generate.Execute(), "generate failed:\n%s", errBuf.String())
			}), "contrast generate needs to succeed for subsequent tests")

			require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			t.Cleanup(cancel)
			require.NoError(t, ct.Kubeclient.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, name))
		})
	}
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.BoolVar(&skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}
