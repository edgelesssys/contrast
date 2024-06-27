// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

///go:build e2e

package policy

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/stretchr/testify/require"
)

const opensslBackend = "openssl-backend"
const opensslFrontend = "openssl-frontend"

var (
	imageReplacementsFile, namespaceFile string
	skipUndeploy                         bool
)

func TestPolicy(t *testing.T) {
	ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, skipUndeploy)

	resources := kuberesource.OpenSSL()

	coordinator := kuberesource.CoordinatorBundle()
	resources = append(resources, coordinator...)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	// initial deployment with pod allowed
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	t.Run("pod cannot join after it was removed from the manifest", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		c := kubeclient.NewForTest(t)

		require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))

		// parse the manifest
		manifestBytes, err := os.ReadFile(ct.WorkDir + "/manifest.json")
		require.NoError(err)
		var m manifest.Manifest
		require.NoError(json.Unmarshal(manifestBytes, &m))
		t.Log("original manifest:", string(manifestBytes))

		// Replace all policy hashes with random ones.
		// This is needed because `ct.Set` fails if the
		// number of policy hashes doesn't match the current deployment
		newPolicies := make(map[manifest.HexString][]string)
		for range m.Policies {
			hash := make([]byte, 32)
			rand.Read(hash)
			newPolicies[manifest.NewHexString(hash)] = []string{""}
		}
		m.Policies = newPolicies

		// write the new manifest
		manifestBytes, err = json.Marshal(m)
		require.NoError(err)
		require.NoError(os.WriteFile(ct.WorkDir+"/manifest.json", manifestBytes, 0o644))
		t.Log("new manifest:", string(manifestBytes))

		// set the new manifest
		ct.Set(t)

		// restart a deployment - this should fail since the manifest disallows the hash
		require.Error(c.RestartDeployment(ctx, ct.Namespace, opensslBackend))
	})
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.BoolVar(&skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}
