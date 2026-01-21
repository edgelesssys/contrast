// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package attestation

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/edgelesssys/contrast/sdk"
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

	// Test that TDX PCK configuration options are validated correctly.
	t.Run("tdx-pck-config", func(t *testing.T) {
		platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
		require.NoError(t, err)
		if !platforms.IsTDX(platform) {
			// This depends on the CI runners being configured with lax settings for memory
			// integrity, dynamic package additions and SMT. This is the result of our currently
			// documented setup:
			// https://github.com/canonical/tdx/blob/1c9ca39/README.md?plain=1#L114.
			t.Skip()
		}

		ct := contrasttest.New(t)

		runtimeHandler, err := manifest.RuntimeHandler(platform)
		require.NoError(t, err)
		resources := kuberesource.CoordinatorBundle()
		resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
		resources = kuberesource.AddPortForwarders(resources)
		ct.Init(t, resources)

		require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
		require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		t.Cleanup(cancel)

		require.NoError(t, ct.Kubeclient.WaitForCoordinator(ctx, ct.Namespace))

		manifestBytes, err := os.ReadFile(ct.WorkDir + "/manifest.json")
		require.NoError(t, err)

		for name, tc := range map[string]struct {
			patchTDXReferenceValues func(*manifest.TDXReferenceValues)
			wantErr                 string
		}{
			"memory-integrity": {
				patchTDXReferenceValues: func(refVal *manifest.TDXReferenceValues) {
					refVal.MemoryIntegrity = true
				},
				wantErr: "PCK extension SGXType",
			},
			"dynamic-platform": {
				patchTDXReferenceValues: func(refVal *manifest.TDXReferenceValues) {
					refVal.StaticPlatform = true
				},
				wantErr: "PCK extension DynamicPlatform",
			},
			"SMT-enabled": {
				patchTDXReferenceValues: func(refVal *manifest.TDXReferenceValues) {
					refVal.SMTDisabled = true
				},
				wantErr: "PCK extension SMTEnabled",
			},
		} {
			t.Run(name, func(t *testing.T) {
				require := require.New(t)
				ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(time.Minute))
				t.Cleanup(cancel)

				var m manifest.Manifest
				require.NoError(json.Unmarshal(manifestBytes, &m))

				for i, tdx := range m.ReferenceValues.TDX {
					tc.patchTDXReferenceValues(&tdx)
					m.ReferenceValues.TDX[i] = tdx
				}

				logger := slog.Default()
				store := fsstore.New(t.TempDir(), logger)
				cache := certcache.NewCachedHTTPSGetter(store, certcache.NeverGCTicker, logger)
				validators, err := sdk.ValidatorsFromManifest(cache, &m, logger)
				require.NoError(err)

				key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				require.NoError(err)
				cfg, err := atls.CreateAttestationClientTLSConfig(ctx, nil, validators, key)
				require.NoError(err)

				require.ErrorContains(ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-coordinator", userapi.Port, func(addr string) error {
					dialer := tls.Dialer{Config: cfg}
					conn, err := dialer.DialContext(ctx, "tcp", addr)
					if err == nil {
						conn.Close()
					}
					return err
				}), tc.wantErr)
			})
		}
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
