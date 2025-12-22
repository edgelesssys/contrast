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
	// Test that the validator fails in case the chip ID in the manifest (AllowedChipIDs) doesn't match.
	t.Run("allowed-chip-ids", func(t *testing.T) {
		platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
		require.NoError(t, err)
		if !platforms.IsSNP(platform) {
			t.Skip()
		}

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

	// Same as above, but for TDX PIIDs.
	t.Run("allowed-piids", func(t *testing.T) {
		platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
		require.NoError(t, err)
		if !platforms.IsTDX(platform) {
			t.Skip()
		}

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

		ct.PatchManifest(t, func(m manifest.Manifest) manifest.Manifest {
			for i := range m.ReferenceValues.TDX {
				m.ReferenceValues.TDX[i].AllowedPIIDs = []manifest.HexString{
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				}
			}
			return m
		})
		require.True(t.Run("set", func(t *testing.T) {
			err := ct.RunSet(t.Context())
			require.ErrorContains(err, "not in allowed PIIDs")
		}), "contrast set should fail due to non-allowed PIID")
	})

	// Test that it is okay to have failing validators as long as one validator passes.
	t.Run("non-matching-validators", func(t *testing.T) {
		platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
		require.NoError(t, err)
		if !platforms.IsSNP(platform) {
			t.Skip()
		}

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

		ct.PatchManifest(t, func(m manifest.Manifest) manifest.Manifest {
			switch platform {
			case platforms.MetalQEMUSNP, platforms.MetalQEMUSNPGPU:
				// Duplicate the first validator.
				m.ReferenceValues.SNP = append(m.ReferenceValues.SNP, m.ReferenceValues.SNP[0])
				// Make the first set of reference values invalid by changing the SVNs.
				m.ReferenceValues.SNP[0].MinimumTCB = manifest.SNPTCB{
					BootloaderVersion: toPtr(manifest.SVN(255)),
					TEEVersion:        toPtr(manifest.SVN(255)),
					SNPVersion:        toPtr(manifest.SVN(255)),
					MicrocodeVersion:  toPtr(manifest.SVN(255)),
				}
			case platforms.MetalQEMUTDX:
				// Duplicate the first validator.
				m.ReferenceValues.TDX = append(m.ReferenceValues.TDX, m.ReferenceValues.TDX[0])
				// Make the first set of reference values invalid by changing the SVNs.
				m.ReferenceValues.TDX[0].MrSeam = manifest.HexString("111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111")
			}
			return m
		})
		require.True(t.Run("set", ct.Set), "set should succeed as long as one validator passes")
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}

func toPtr[T any](t T) *T {
	return &t
}
