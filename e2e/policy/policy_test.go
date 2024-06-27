// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package policy

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/stretchr/testify/require"
)

const opensslBackend = "openssl-backend"

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
		t.Log("original manifest:", manifestBytes)

		// remove the openssl backend from the policy hashes
		newPolicies := make(map[manifest.HexString][]string)
		for policyHash := range m.Policies {
			if slices.Contains(m.Policies[policyHash], opensslBackend) {
				continue // skip the policy
			}
			newPolicies[policyHash] = m.Policies[policyHash]
		}
		m.Policies = newPolicies
		manifestBytes, err = json.Marshal(m)
		require.NoError(err)
		require.NoError(os.WriteFile(ct.WorkDir+"/manifest.json", manifestBytes, 0o644))
		t.Log("new manifest:", manifestBytes)

		// set the new manifest
		ct.Set(t)

		// restart the deployment - this should fail since the manifest disallows the hash
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
