// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProcessorConfig(t *testing.T) {
	tests := map[string]struct {
		data         map[string]string
		wantInsecure bool
		wantErr      bool
	}{
		"missing": {},
		"insecure": {
			data: map[string]string{
				initdataProcessorConfigKey: `{"insecure": true}`,
			},
			wantInsecure: true,
		},
		"secure": {
			data: map[string]string{
				initdataProcessorConfigKey: `{"insecure": false}`,
			},
		},
		"invalid": {
			data: map[string]string{
				initdataProcessorConfigKey: `{insecure: true}`,
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			config, err := parseProcessorConfig(tc.data)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantInsecure, config.Insecure)
		})
	}
}
