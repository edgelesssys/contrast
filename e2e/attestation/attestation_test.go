// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package attestation

import (
	"flag"
	"os"
	"testing"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/require"
)

func TestAttestation(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)

	require := require.New(t)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(err)
	resources := kuberesource.CoordinatorBundle()
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)
	ct.Init(t, resources)

	require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	// Test that the validator fails in case the chip ID in the manifest (AllowedChipIDs) doesn't match.
	t.Run("allowed-chip-ids", func(t *testing.T) {
		ct.PatchManifest(t, func(m manifest.Manifest) manifest.Manifest {
			for i := range m.ReferenceValues.SNP {
				m.ReferenceValues.SNP[i].AllowedChipIDs = []manifest.HexString{
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				}
			}
			return m
		})
		require.True(t.Run("set", func(t *testing.T) {
			err := ct.RunSet(t.Context())
			require.ErrorContains(err, "not in allowed chip IDs")
		}), "contrast set should fail due to non-allowed chip ID")
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
