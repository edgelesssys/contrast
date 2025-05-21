// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package genpolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenpolicyLogPrefixReg(t *testing.T) {
	testCases := []struct {
		logLine              string
		wantMatch            bool
		wantLevel            string
		wantPosition         string
		wantMessage          string
		wantNoPrefixErrMatch bool
	}{
		{
			logLine:      `[2024-06-26T09:09:40Z INFO  genpolicy::registry] ============================================`,
			wantMatch:    true,
			wantLevel:    "INFO",
			wantPosition: "genpolicy::registry",
			wantMessage:  "============================================",
		},
		{
			logLine:      `[2024-06-26T09:09:40Z INFO  genpolicy::registry] Pulling manifest and config for "mcr.microsoft.com/oss/kubernetes/pause:3.6"`,
			wantMatch:    true,
			wantLevel:    "INFO",
			wantPosition: "genpolicy::registry",
			wantMessage:  `Pulling manifest and config for "mcr.microsoft.com/oss/kubernetes/pause:3.6"`,
		},
		{
			logLine:      `[2024-06-26T09:09:41Z INFO  genpolicy::registry] Using cache file`,
			wantMatch:    true,
			wantLevel:    "INFO",
			wantPosition: "genpolicy::registry",
			wantMessage:  "Using cache file",
		},
		{
			logLine:      `[2024-06-26T09:09:41Z INFO  genpolicy::registry] dm-verity root hash: 1e306eb31693964ac837ac74bc57b50c87c549f58b0da2789e3915f923f21b81`,
			wantMatch:    true,
			wantLevel:    "INFO",
			wantPosition: "genpolicy::registry",
			wantMessage:  "dm-verity root hash: 1e306eb31693964ac837ac74bc57b50c87c549f58b0da2789e3915f923f21b81",
		},
		{
			logLine:      `[2024-06-26T09:09:44Z WARN  genpolicy::registry] Using cache file`,
			wantMatch:    true,
			wantLevel:    "WARN",
			wantPosition: "genpolicy::registry",
			wantMessage:  "Using cache file",
		},
		{
			logLine:              `thread 'main' panicked at src/registry.rs:131:17:`,
			wantMatch:            false,
			wantNoPrefixErrMatch: true,
		},
		{
			logLine:   `Success!"`,
			wantMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			assert := assert.New(t)

			match := genpolicyLogPrefixReg.FindStringSubmatch(tc.logLine)

			if !tc.wantMatch {
				assert.Nil(match)
				if tc.wantNoPrefixErrMatch {
					assert.True(errorMessage.MatchString(tc.logLine))
				}
				return
			}
			assert.Len(match, 4)
			assert.Equal(tc.wantLevel, match[1])
			assert.Equal(tc.wantPosition, match[2])
			assert.Equal(tc.wantMessage, match[3])
		})
	}
}
