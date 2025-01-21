// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

//go:embed golden/defaultEnvoy.json
var defaultEnvoyConfig []byte

func TestCompareEnvoyConfigToGolden(t *testing.T) {
	require := require.New(t)

	config, err := ParseProxyConfig("", "", "")
	require.NoError(err)

	testCases := map[string]struct {
		pConfig ProxyConfig
	}{
		"success": {
			pConfig: config,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			configJSON, err := tc.pConfig.ToEnvoyConfig()
			require.NoError(err)
			assert.JSONEq(string(defaultEnvoyConfig), string(configJSON))
		})
	}
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
