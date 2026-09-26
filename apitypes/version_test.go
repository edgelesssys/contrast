// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apitypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAPIVersion(t *testing.T) {
	for version, want := range map[string]int{
		"v1":  1,
		"v2":  2,
		"v10": 10,
	} {
		t.Run(version, func(t *testing.T) {
			got, err := ParseAPIVersion(version)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}

	for _, version := range []string{"", "1", "v", "v0", "v01", "V1", "v-1", "v+1", "v1.2", "v1x", "vv1"} {
		t.Run("invalid_"+version, func(t *testing.T) {
			_, err := ParseAPIVersion(version)
			require.Error(t, err)
		})
	}
}
