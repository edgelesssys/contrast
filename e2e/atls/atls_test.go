// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package atls

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/idblock"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/memstore"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/google/go-sev-guest/abi"
	"github.com/stretchr/testify/require"
)

// TestSNPValidators runs e2e tests for the atls layer.
func TestSNPValidators(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)

	if platform != platforms.AKSCloudHypervisorSNP &&
		platform != platforms.K3sQEMUSNP &&
		platform != platforms.K3sQEMUSNPGPU &&
		platform != platforms.MetalQEMUSNPGPU &&
		platform != platforms.MetalQEMUSNP {
		t.Skip("Skipping test, test can only be run on SEV-SNP-based platforms")
	}

	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	coordinator := kuberesource.CoordinatorBundle()

	coordinator = kuberesource.PatchRuntimeHandlers(coordinator, runtimeHandler)

	coordinator = kuberesource.AddPortForwarders(coordinator)

	ct.Init(t, coordinator)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	require.True(t, t.Run("verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	var coordPolicyHashBytes []byte
	var manifestParsed manifest.Manifest
	require.True(t, t.Run("prepare atls tests", func(t *testing.T) {
		require := require.New(t)
		manifestBytes, err := os.ReadFile(path.Join(ct.WorkDir, "manifest.json"))
		require.NoError(err, "reading manifest file")
		require.NoError(json.Unmarshal(manifestBytes, &manifestParsed))
		require.NoError(manifestParsed.Validate())

		coordPolicyHash, err := manifestParsed.CoordinatorPolicyHash()
		require.NoError(err, "getting coordinator policy hash")
		coordPolicyHashBytes, err = coordPolicyHash.Bytes()
		require.NoError(err, "converting coordinator policy hash to bytes")
	}))

	idKeyDigestReplace := func(measurement [48]byte, guestPolicy abi.SnpPolicy, opt *manifest.SNPValidatorOptions) error {
		// Generate static public IDKey based on the launch digest and guest policy.
		_, authBlk, err := idblock.IDBlocksFromLaunchDigest(measurement, guestPolicy)
		if err != nil {
			return fmt.Errorf("failed to generate ID blocks: %w", err)
		}
		idKeyBytes, err := authBlk.IDKey.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal IDKey: %w", err)
		}
		idKeyHash := sha512.Sum384(idKeyBytes)
		opt.ValidateOpts.TrustedIDKeyHashes = [][]byte{idKeyHash[:]}
		return nil
	}

	testCases := map[string]struct {
		manifestModifyFunc func(*manifest.SNPValidatorOptions) error
		wantError          bool
	}{
		"default values": {
			manifestModifyFunc: func(_ *manifest.SNPValidatorOptions) error {
				return nil
			},
			wantError: false,
		},
		"platformInfo flip SMT to false": {
			manifestModifyFunc: func(opt *manifest.SNPValidatorOptions) error {
				if opt.ValidateOpts.PlatformInfo.SMTEnabled == false {
					return fmt.Errorf("SMT must be disabled by default")
				}
				opt.ValidateOpts.PlatformInfo.SMTEnabled = false
				return nil
			},
			wantError: true,
		},
		"idKeyDigestReplace flip Debug to true": {
			manifestModifyFunc: func(opt *manifest.SNPValidatorOptions) error {
				if opt.ValidateOpts.GuestPolicy.Debug == true {
					return fmt.Errorf("Debug must be disabled by default")
				}
				// All fields in GuestPolicy are primitive types, so this copies whole struct.
				guestPolicy := opt.ValidateOpts.GuestPolicy
				guestPolicy.Debug = true
				return idKeyDigestReplace([48]byte(opt.ValidateOpts.Measurement), guestPolicy, opt)
			},
			wantError: true,
		},
		"idKeyDigestReplace flip bits in measurement": {
			manifestModifyFunc: func(opt *manifest.SNPValidatorOptions) error {
				measurement := opt.ValidateOpts.Measurement
				measurement[0] ^= 0xFF
				return idKeyDigestReplace([48]byte(measurement), opt.ValidateOpts.GuestPolicy, opt)
			},
			wantError: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, ct.Kubeclient.WithForwardedPort(ctx, ct.Namespace, "port-forwarder-coordinator", "1313", func(addr string) error {
				assert := require.New(t)
				require := require.New(t)

				kdsCache := memstore.New[string, []byte]()
				kdsGetter := certcache.NewCachedHTTPSGetter(kdsCache, certcache.NeverGCTicker, logger.WithGroup("kds-getter"))
				opts, err := manifestParsed.SNPValidateOpts(kdsGetter)
				require.NoError(err, "getting SNP validate options")
				var validators []atls.Validator
				for i, opt := range opts {
					opt.ValidateOpts.HostData = coordPolicyHashBytes
					assert.NoError(tc.manifestModifyFunc(&opt))
					name := fmt.Sprintf("snp-%d-%s", i, strings.TrimPrefix(opt.VerifyOpts.Product.Name.String(), "SEV_PRODUCT_"))
					validators = append(validators, snp.NewValidator(opt.VerifyOpts, opt.ValidateOpts, logger.WithGroup("validator"), name))
				}

				dialer := dialer.New(atls.NoIssuer, validators, atls.NoMetrics, nil, logger)
				conn, err := dialer.Dial(ctx, addr)
				require.NoError(err, "dialing coordinator")
				defer conn.Close()

				client := userapi.NewUserAPIClient(conn)
				_, err = client.GetManifests(ctx, &userapi.GetManifestsRequest{})
				if tc.wantError {
					assert.Error(err, "getting manifests")
					return nil
				}
				assert.NoError(err, "getting manifests")

				return nil
			}))
		})
	}
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}
