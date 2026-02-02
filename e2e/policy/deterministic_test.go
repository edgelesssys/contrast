// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package policy

import (
	"encoding/json"
	"fmt"
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
	require := require.New(t)
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(err)

	// create K8s resources
	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(err)
	resources := kuberesource.OpenSSL()
	coordinatorBundle := kuberesource.CoordinatorBundle() // only required because ct.Generate requires the coordinator hash file to be present
	resources = append(resources, coordinatorBundle...)
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	unstructuredResources, err := kuberesource.ResourcesToUnstructured(resources)
	require.NoError(err)
	resourcesBytes, err := kuberesource.EncodeUnstructured(unstructuredResources)
	require.NoError(err)

	// generate policy 5 times and check if the policy hash is the same
	var expectedPolicies map[manifest.HexString]manifest.PolicyEntry
	for i := range 5 {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			policies := runGenerate(t, resourcesBytes)

			// verify that policies are deterministic
			if expectedPolicies != nil {
				require.Equal(expectedPolicies, policies, "expected deterministic policy generation")
			} else {
				expectedPolicies = policies // only set policies on the first run
			}
		})
	}
	t.Log("Policies are deterministic")
}

func runGenerate(t *testing.T, resources []byte) map[manifest.HexString]manifest.PolicyEntry {
	require := require.New(t)
	ct := contrasttest.New(t)

	require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yml"), resources, 0o644)) // reset file for each run
	require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	manifestBytes, err := os.ReadFile(ct.ManifestPath())
	require.NoError(err)

	var m manifest.Manifest
	require.NoError(json.Unmarshal(manifestBytes, &m))

	return m.Policies
}
