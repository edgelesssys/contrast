// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrustedRoots(t *testing.T) {
	roots, err := amdTrustedRootCerts(Milan)
	assert.NoError(t, err)
	assert.Contains(t, roots, "Milan")
	assert.NotContains(t, roots, "Genoa")

	roots, err = amdTrustedRootCerts(Genoa)
	assert.NoError(t, err)
	assert.NotContains(t, roots, "Milan")
	assert.Contains(t, roots, "Genoa")
}

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
