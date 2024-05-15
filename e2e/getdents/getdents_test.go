// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package getdents

import (
	"bytes"
	"context"
	"flag"
	"io"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

const (
	getdent = "getdents-tester"
)

var imageReplacements map[string]string

func TestGetDEnts(t *testing.T) {
	ct := contrasttest.New(t, imageReplacements)

	resources, err := kuberesource.GetDEnts()
	require.NoError(t, err)

	ct.Init(t, resources)

	// Call generate to patch the runtime class.
	require.True(t, t.Run("generate", func(t *testing.T) {
		require := require.New(t)
		args := append([]string{"--workspace-dir", ct.WorkDir}, path.Join(ct.WorkDir, "resources.yaml"))
		generate := cmd.NewGenerateCmd()
		generate.Flags().String("workspace-dir", "", "") // Make generate aware of root flags
		generate.SetArgs(args)
		generate.SetOut(io.Discard)
		errBuf := &bytes.Buffer{}
		generate.SetErr(errBuf)

		require.NoError(generate.Execute())
	}), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	// This tests an upstream bug in TarFS where a 'getdents' call hangs in an infinite loop when too many files are in a directory.
	// If the 'find' command results in a timeout, the test fails.
	t.Run("call find on large folder", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, getdent))

		pods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, getdent)
		require.NoError(err)
		require.Len(pods, 1, "pod not found: %s/%s", ct.Namespace, getdent)

		stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, pods[0].Name, []string{"/bin/sh", "-c", "find /toomany | wc -l"})
		require.NoError(err, "stderr: %q", stderr)
		require.Equal(stdout, "10001\n", "output: %s", stdout)
	})
}

func TestMain(m *testing.M) {
	flag.Parse()

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("could not open image definition file %q: %v", flag.Arg(0), err)
	}
	imageReplacements, err = kuberesource.ImageReplacementsFromFile(f)
	if err != nil {
		log.Fatalf("could not parse image definition file %q: %v", flag.Arg(0), err)
	}

	os.Exit(m.Run())
}
