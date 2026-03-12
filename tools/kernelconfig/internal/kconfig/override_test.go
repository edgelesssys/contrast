// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kconfig

import (
	_ "embed"
	"testing"

	"github.com/edgelesssys/contrast/tools/kernelconfig/internal/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/expected-config
	expectedConf []byte
	//go:embed testdata/expected-config-nvidia-gpu
	expectedConfGPU []byte
)

func TestOverrideConfig(t *testing.T) {
	testCases := map[string]struct {
		base  []byte
		isGPU bool
		want  []byte
	}{
		"non-gpu": {
			base: base.BaseConfig,
			want: expectedConf,
		},
		"gpu": {
			base:  base.BaseConfigGPU,
			isGPU: true,
			want:  expectedConfGPU,
		},
	}
	for config, tc := range testCases {
		t.Run(config, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			cfg, err := OverrideConfig(tc.base, tc.isGPU)
			require.NoError(err)

			configBytes := cfg.Marshal()
			assert.Equal(tc.want, configBytes)
		})
	}
}
