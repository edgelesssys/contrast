// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestManifestSNP() *Manifest {
	return &Manifest{
		Policies: map[HexString]PolicyEntry{
			HexString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"): {
				Role:             "coordinator",
				SANs:             []string{"example.com", "*"},
				WorkloadSecretID: "foo",
			},
		},
		ReferenceValues: ReferenceValues{
			SNP: []SNPReferenceValues{
				{
					MinimumTCB: SNPTCB{
						BootloaderVersion: toPtr(SVN(2)),
						TEEVersion:        toPtr(SVN(2)),
						SNPVersion:        toPtr(SVN(2)),
						MicrocodeVersion:  toPtr(SVN(2)),
					},
					ProductName:        "Milan",
					TrustedMeasurement: HexString("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"),
					GuestPolicy: abi.SnpPolicy{
						SMT: true,
					},
				},
			},
		},
		WorkloadOwnerKeyDigests: []HexString{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		SeedshareOwnerPubKeys:   []HexString{"3082020a0282020100bc51b1c9cb50002ae95a7ad98569506aadceabd0bb9ee576ad29ab3baecd60a604eee8882323fa8a7b6f4b21cea80bd4d794bf31fd7a5d6dfeb4afa6de30a834b87aa7f90e5ee11683b2f903b393174c436107f7b22467d0f5cee09c43eab28bcbec0137e165d8d34da66f9fc8294d60ebafbf38bf3e0e4dfcf7da84b24d6eaf1b8c4a579d2ec6ab1c280a7ec50854f2269ef8e31e1e2f1a1d24dd3ca8eaa5728c4f59fdf840899dc44bac0749b3d294a3c7446e2859db55cb93e6bf8ac8995665137f74e9898dad7a9f52d8527b16308e422685ed6ba221b6a266728a7cc11403d37a2f1be923685637eabafa4b0bf6095edc6908adf1b623450555ead19f9431d9a72730228379d5711475434e4696eb650469ad981cc0ced54c84909c36dc7e38d1ffda3aad63d2b6841bb2cee07c85116cd8139e528a3ca78a0d301f2deed86f0b7bbc6bcd863594e3ba67d86178db9f661d0aa25965ad05c608b8c7eadb5853284aadf2474b8a3dbbc23ac2992c11b283808108ffd59cf08687ac9a0ef20eb8ca3ec090a3a532ad305203b877a20c0da401386070dc97d2820b97d67eedc21c6a938bb368a7d1b4e9433abc3304d2f928df22d5d3b3bc0d593b1dcc84671809538b5da667a5f2ce89714c39d9c6aed34de3605fe79e9018de14b262ab1c63bb7d24c3ec51dac00b72be7266e6e1e221da1e50e9ea21aba44550c48bcdeb0203010001"},
	}
}

func newTestManifestTDX() *Manifest {
	m := newTestManifestSNP()
	m.ReferenceValues = ReferenceValues{
		TDX: []TDXReferenceValues{
			{
				MrTd: HexString("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
				Rtrms: [4]HexString{
					"555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555",
					"666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666",
					"777777777777777777777777777777777777777777777777777777777777777777777777777777777777777777777777",
					"888888888888888888888888888888888888888888888888888888888888888888888888888888888888888888888888",
				},
				MinimumQeSvn:     toPtr(uint16(5)),
				MinimumPceSvn:    toPtr(uint16(6)),
				MinimumTeeTcbSvn: HexString("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"),
				MrSeam:           HexString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
				TdAttributes:     HexString("3333333333333333"),
				Xfam:             HexString("4444444444444444"),
			},
		},
	}
	return m
}

func TestValidate(t *testing.T) {
	testCases := map[string]struct {
		m       *Manifest
		mutate  func(*Manifest)
		wantErr bool
	}{
		"valid SNP manifest": {
			m: newTestManifestSNP(),
		},
		"valid TDX manifest": {
			m: newTestManifestTDX(),
		},
		"invalid policy hash": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.Policies[HexString("invalid")] = PolicyEntry{}
			},
			wantErr: true,
		},
		"invalid policy hash length": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.Policies[HexString("")] = PolicyEntry{}
			},
			wantErr: true,
		},
		"invalid policy role": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.Policies[HexString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")] = PolicyEntry{
					Role: "invalid",
				}
			},
			wantErr: true,
		},
		"trusted measurement empty": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.SNP[0].TrustedMeasurement = HexString("")
			},
			wantErr: true,
		},
		"invalid workload owner key digest": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.WorkloadOwnerKeyDigests = []HexString{"invalid"}
			},
			wantErr: true,
		},
		"invalid workload owner key digest length": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.WorkloadOwnerKeyDigests = []HexString{"aaaa"}
			},
			wantErr: true,
		},
		"invalid seedshare owner public key": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.SeedshareOwnerPubKeys = []HexString{"invalid"}
			},
			wantErr: true,
		},
		"snp bootloader version empty": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.SNP[0].MinimumTCB.BootloaderVersion = nil
			},
			wantErr: true,
		},
		"snp tee version empty": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.SNP[0].MinimumTCB.TEEVersion = nil
			},
			wantErr: true,
		},
		"snp snp version empty": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.SNP[0].MinimumTCB.SNPVersion = nil
			},
			wantErr: true,
		},
		"snp microcode version empty": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.SNP[0].MinimumTCB.MicrocodeVersion = nil
			},
			wantErr: true,
		},
		"snp unknown product name": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.SNP[0].ProductName = "unknown"
			},
			wantErr: true,
		},
		"snp guest policy smt not set": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.SNP[0].GuestPolicy.SMT = false
			},
			wantErr: true,
		},
		"no reference values": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.ReferenceValues = ReferenceValues{}
			},
			wantErr: true,
		},
		"tdx mr td empty": {
			m: newTestManifestTDX(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.TDX[0].MrTd = ""
			},
			wantErr: true,
		},
		"tdx rtrms empty": {
			m: newTestManifestTDX(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.TDX[0].Rtrms = [4]HexString{}
			},
			wantErr: true,
		},
		"tdx minimum qe svn empty": {
			m: newTestManifestTDX(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.TDX[0].MinimumQeSvn = nil
			},
			wantErr: true,
		},
		"tdx minimum pce svn empty": {
			m: newTestManifestTDX(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.TDX[0].MinimumPceSvn = nil
			},
			wantErr: true,
		},
		"tdx minimum tee tcb svn empty": {
			m: newTestManifestTDX(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.TDX[0].MinimumTeeTcbSvn = ""
			},
			wantErr: true,
		},
		"tdx mr seam empty": {
			m: newTestManifestTDX(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.TDX[0].MrSeam = ""
			},
			wantErr: true,
		},
		"tdx td attributes empty": {
			m: newTestManifestTDX(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.TDX[0].TdAttributes = ""
			},
			wantErr: true,
		},
		"tdx xfam empty": {
			m: newTestManifestTDX(),
			mutate: func(m *Manifest) {
				m.ReferenceValues.TDX[0].Xfam = ""
			},
			wantErr: true,
		},
		"no coordinator policy": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				for k := range m.Policies {
					p := m.Policies[k]
					p.Role = RoleNone
					m.Policies[k] = p
				}
			},
			wantErr: true,
		},
		"two coordinator policies": {
			m: newTestManifestSNP(),
			mutate: func(m *Manifest) {
				m.Policies[HexString("2bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")] = PolicyEntry{
					Role: "coordinator",
				}
			},
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			if tc.mutate != nil {
				tc.mutate(tc.m)
			}

			if tc.wantErr {
				err := tc.m.Validate()
				t.Log(err)
				assert.Error(err)
				return
			}
			assert.NoError(tc.m.Validate())
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

func TestSNPValidateOpts(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	m, err := Default(platforms.AKSCloudHypervisorSNP)
	t.Log(err)
	require.NoError(err)

	m.Policies = map[HexString]PolicyEntry{
		HexString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"): {
			Role: RoleCoordinator,
		},
	}

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

func TestHexString(t *testing.T) {
	testCases := []struct {
		b []byte
		s string
	}{
		{b: []byte{0x00}, s: "00"},
		{b: []byte{0x01}, s: "01"},
		{b: []byte{0x0f}, s: "0f"},
		{b: []byte{0x10}, s: "10"},
		{b: []byte{0x11}, s: "11"},
		{b: []byte{0xff}, s: "ff"},
		{b: []byte{0x00, 0x01}, s: "0001"},
	}

	t.Run("Bytes", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.s, func(t *testing.T) {
				assert := assert.New(t)
				hexString := NewHexString(tc.b)
				assert.Equal(tc.s, hexString.String())
				b, err := hexString.Bytes()
				assert.NoError(err)
				assert.Equal(tc.b, b)
			})
		}
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.s, func(t *testing.T) {
				assert := assert.New(t)
				hexString := NewHexString(tc.b)
				enc, err := json.Marshal(hexString)
				assert.NoError(err)
				assert.Equal(fmt.Sprintf("\"%s\"", tc.s), string(enc))
			})
		}
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.s, func(t *testing.T) {
				assert := assert.New(t)
				var hexString HexString
				err := json.Unmarshal(fmt.Appendf(nil, "\"%s\"", tc.s), &hexString)
				assert.NoError(err)
				assert.Equal(tc.s, hexString.String())
			})
		}
	})

	t.Run("invalid hexstring", func(t *testing.T) {
		assert := assert.New(t)
		hexString := HexString("invalid")
		_, err := hexString.Bytes()
		assert.Error(err)
	})
}

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

func toPtr[T any](v T) *T {
	return &v
}

func TestExpectedMissingReferenceValues(t *testing.T) {
	testCases := map[string]struct {
		m    *Manifest
		want bool
	}{
		"tdx only expected validation errors": {
			m: func() *Manifest {
				m := newTestManifestTDX()
				m.ReferenceValues.TDX[0].MrSeam = ""
				return m
			}(),
			want: true,
		},
		"tdx with unexpected validation errors": {
			m: func() *Manifest {
				m := newTestManifestTDX()
				m.ReferenceValues.TDX[0].MrTd = ""
				return m
			}(),
			want: false,
		},
		"snp only expected validation errors": {
			m: func() *Manifest {
				m := newTestManifestSNP()
				m.ReferenceValues.SNP[0].MinimumTCB.TEEVersion = nil
				return m
			}(),
			want: true,
		},
		"snp with unexpected validation errors": {
			m: func() *Manifest {
				m := newTestManifestSNP()
				m.ReferenceValues.SNP[0].TrustedMeasurement = ""
				return m
			}(),
			want: false,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			var ve *ValidationError
			err := tc.m.Validate()
			assert.Error(err)
			require.ErrorAs(t, err, &ve)
			assert.Equal(tc.want, ve.OnlyExpectedMissingReferenceValues())
		})
	}
}
