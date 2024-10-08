// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"encoding/base64"
	"strconv"
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/google/go-sev-guest/kds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHexStrings(t *testing.T) {
	testCases := []struct {
		hs      HexStrings
		bs      [][]byte
		wantErr bool
	}{
		{
			hs: HexStrings{"00", "01"},
			bs: [][]byte{{0x00}, {0x01}},
		},
		{
			hs: HexStrings{"00", "01", "0f", "10", "11", "ff"},
			bs: [][]byte{{0x00}, {0x01}, {0x0f}, {0x10}, {0x11}, {0xff}},
		},
		{
			hs:      HexStrings{"00", "01", "0f", "10", "11", "ff", "invalid"},
			wantErr: true,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert := assert.New(t)
			bs, err := tc.hs.ByteSlices()
			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.Equal(tc.bs, bs)
		})
	}
}

func TestPolicy(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert := assert.New(t)

		policy := []byte("test-policy")
		expectedHash := HexString("48a7cea3db9b9bf087e58bdff6e7a4260a0227b90ba0fceb97060a3c76e004e1")

		annotation := base64.StdEncoding.EncodeToString(policy)
		p, err := NewPolicyFromAnnotation([]byte(annotation))

		assert.NoError(err)
		assert.Equal(policy, p.Bytes())
		assert.Equal(expectedHash, p.Hash())
	})
	t.Run("invalid", func(t *testing.T) {
		assert := assert.New(t)

		annotation := "invalid"
		_, err := NewPolicyFromAnnotation([]byte(annotation))

		assert.Error(err)
	})
}

func TestValidate(t *testing.T) {
	mnf, err := Default(platforms.AKSCloudHypervisorSNP)
	require.NoError(t, err)

	testCases := []struct {
		m       *Manifest
		wantErr bool
	}{
		{
			m: mnf,
		},
		{
			m: &Manifest{
				Policies:        map[HexString]PolicyEntry{HexString(""): {}},
				ReferenceValues: mnf.ReferenceValues,
			},
			wantErr: true,
		},
		{
			m: &Manifest{
				Policies: map[HexString]PolicyEntry{HexString(""): {}},
				ReferenceValues: ReferenceValues{
					SNP: []SNPReferenceValues{
						{
							MinimumTCB:         mnf.ReferenceValues.SNP[0].MinimumTCB,
							TrustedMeasurement: "",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			m: &Manifest{
				ReferenceValues:         mnf.ReferenceValues,
				WorkloadOwnerKeyDigests: []HexString{HexString("")},
			},
			wantErr: true,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert := assert.New(t)

			if tc.wantErr {
				assert.Error(tc.m.Validate())
				return
			}
			assert.NoError(tc.m.Validate())
		})
	}
}

func TestAKSValidateOpts(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	m, err := Default(platforms.AKSCloudHypervisorSNP)
	require.NoError(err)

	opts, err := m.SNPValidateOpts(nil)
	require.NoError(err)
	require.Len(opts, 1)

	tcb := m.ReferenceValues.SNP[0].MinimumTCB
	assert.NotNil(tcb.BootloaderVersion)
	assert.NotNil(tcb.TEEVersion)
	assert.NotNil(tcb.SNPVersion)
	assert.NotNil(tcb.MicrocodeVersion)

	trustedMeasurement, err := m.ReferenceValues.SNP[0].TrustedMeasurement.Bytes()
	assert.NoError(err)

	assert.Equal(trustedMeasurement, opts[0].ValidateOpts.Measurement)

	tcbParts := kds.TCBParts{
		BlSpl:    tcb.BootloaderVersion.UInt8(),
		TeeSpl:   tcb.TEEVersion.UInt8(),
		SnpSpl:   tcb.SNPVersion.UInt8(),
		UcodeSpl: tcb.MicrocodeVersion.UInt8(),
	}
	assert.Equal(tcbParts, opts[0].ValidateOpts.MinimumTCB)
	assert.Equal(tcbParts, opts[0].ValidateOpts.MinimumLaunchTCB)
}

func TestTrustedRoots(t *testing.T) {
	roots, err := trustedRoots(Milan)
	assert.NoError(t, err)
	assert.Contains(t, roots, "Milan")
	assert.NotContains(t, roots, "Genoa")

	roots, err = trustedRoots(Genoa)
	assert.NoError(t, err)
	assert.NotContains(t, roots, "Milan")
	assert.Contains(t, roots, "Genoa")
}
