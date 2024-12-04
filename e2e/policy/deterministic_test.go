// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package policy

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

func TestDeterminsticPolicyGeneration(t *testing.T) {
	platform, err := platforms.FromString(platformStr)
	require := require.New(t)
	require.NoError(err)
	skipUndeploy := true // doesn't matter, because we don't deploy
	ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, platform, skipUndeploy)

	// create K8s resources
	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(err)
	resources := kuberesource.OpenSSL()
	coordinatorBundle := kuberesource.CoordinatorBundle()
	resources = append(resources, coordinatorBundle...)
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)
	buf, err := kuberesource.EncodeUnstructured(unstructuredResources)
	require.NoError(err)

	// generate policy 5 times and check if the policy hash is the same
	var expectedPolicies map[manifest.HexString]manifest.PolicyEntry
	for i := range 5 {
		t.Log("Generate run", i)
		require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yml"), buf, 0o644)) // reset file for each run
		require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
		manifestBytes, err := os.ReadFile(ct.WorkDir + "/manifest.json")
		require.NoError(err)

		// verify that policies are deterministic
		var m manifest.Manifest
		require.NoError(json.Unmarshal(manifestBytes, &m))
		if expectedPolicies != nil {
			require.Equal(expectedPolicies, m.Policies, "expected deterministic policy generation")
		} else {
			expectedPolicies = m.Policies // only set policies on the first run
		}
	}
	t.Log("Policies are deterministic")
}
