// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/constants"
	"github.com/stretchr/testify/require"
)

func TestConfigHasKataSection(t *testing.T) {
	// This is a regression test that ensures the `agent.kata` section is not optimized away. Empty
	// section and no section are handled differently by Kata, so we make sure that this section is
	// always present.
	for _, platform := range platforms.All() {
		t.Run(platform.String(), func(t *testing.T) {
			require := require.New(t)
			cfg, err := constants.KataRuntimeConfig("/", platforms.AKSPeerSNP, "", false)
			require.NoError(err)
			configBytes, err := cfg.Marshal()
			require.NoError(err)
			require.Contains(string(configBytes), "[Agent.kata]")
		})
	}
}
