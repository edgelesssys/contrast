// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/google/go-sev-guest/kds"
	"github.com/stretchr/testify/assert"
)

func TestSVN(t *testing.T) {
	testCases := []struct {
		enc     string
		dec     SVN
		wantErr bool
	}{
		{enc: "0", dec: 0},
		{enc: "1", dec: 1},
		{enc: "255", dec: 255},
		{enc: "256", dec: 0, wantErr: true},
		{enc: "-1", dec: 0, wantErr: true},
	}

	t.Run("MarshalJSON", func(t *testing.T) {
		for _, tc := range testCases {
			if tc.wantErr {
				continue
			}
			t.Run(tc.enc, func(t *testing.T) {
				assert := assert.New(t)
				enc, err := json.Marshal(tc.dec)
				assert.NoError(err)
				assert.Equal(tc.enc, string(enc))
			})
		}
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.enc, func(t *testing.T) {
				assert := assert.New(t)
				var dec SVN
				err := json.Unmarshal([]byte(tc.enc), &dec)
				if tc.wantErr {
					assert.Error(err)
					return
				}
				assert.NoError(err)
				assert.Equal(tc.dec, dec)
			})
		}
	})
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
	testCases := []struct {
		tcb     SNPTCB
		tm      HexString
		wantErr bool
	}{
		{
			tcb: SNPTCB{
				BootloaderVersion: toPtr(SVN(0)),
				TEEVersion:        toPtr(SVN(1)),
				SNPVersion:        toPtr(SVN(2)),
				MicrocodeVersion:  toPtr(SVN(3)),
			},
			tm: HexString("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			tcb: SNPTCB{
				BootloaderVersion: toPtr(SVN(0)),
				TEEVersion:        toPtr(SVN(1)),
				SNPVersion:        toPtr(SVN(2)),
				MicrocodeVersion:  toPtr(SVN(3)),
			},
			tm: HexString(""),
		},
		{
			tcb:     SNPTCB{},
			wantErr: true,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert := assert.New(t)

			mnfst := Manifest{
				ReferenceValues: ReferenceValues{
					SNP:                SNPReferenceValues{MinimumTCB: tc.tcb},
					TrustedMeasurement: tc.tm,
				},
			}

			opts, err := mnfst.SNPValidateOpts()
			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			assert.NotNil(tc.tcb.BootloaderVersion)
			assert.NotNil(tc.tcb.TEEVersion)
			assert.NotNil(tc.tcb.SNPVersion)
			assert.NotNil(tc.tcb.MicrocodeVersion)

			trustedMeasurement, err := tc.tm.Bytes()
			assert.NoError(err)
			if len(tc.tm.String()) == 0 {
				assert.Equal(make([]byte, 48), opts.Measurement)
			} else {
				assert.Equal(trustedMeasurement, opts.Measurement)
			}

			tcbParts := kds.TCBParts{
				BlSpl:    tc.tcb.BootloaderVersion.UInt8(),
				TeeSpl:   tc.tcb.TEEVersion.UInt8(),
				SnpSpl:   tc.tcb.SNPVersion.UInt8(),
				UcodeSpl: tc.tcb.MicrocodeVersion.UInt8(),
			}
			assert.Equal(tcbParts, opts.MinimumTCB)
			assert.Equal(tcbParts, opts.MinimumLaunchTCB)
		})
	}
}
