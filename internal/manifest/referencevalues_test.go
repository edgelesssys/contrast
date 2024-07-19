// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"encoding/json"
	"fmt"
	"testing"

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
				err := json.Unmarshal([]byte(fmt.Sprintf("\"%s\"", tc.s)), &hexString)
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
