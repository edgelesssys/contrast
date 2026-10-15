// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"testing"

	"github.com/edgelesssys/contrast/sdk"
	"github.com/stretchr/testify/require"
)

func TestCheckMinimumAPIVersion(t *testing.T) {
	for name, tc := range map[string]struct {
		manifestPin string
		flagPin     string
		wantErr     bool
	}{
		"no pin":          {},
		"pin in manifest": {manifestPin: "v1", wantErr: true},
		"pin via flag":    {flagPin: "v1", wantErr: true},
		"both":            {manifestPin: "v1", flagPin: "v2", wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			err := checkMinimumAPIVersion(tc.manifestPin, tc.flagPin)
			if tc.wantErr {
				require.ErrorIs(t, err, sdk.ErrMinimumAPIVersionUnmet)
				return
			}
			require.NoError(t, err)
		})
	}
}
