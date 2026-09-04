// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apiv1

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConstructReportDataGolden pins the byte layout of the report data digest.
//
// The digest is baked into attestation reports and reproduced independently by every client,
// so any change here silently breaks verification against already-deployed Coordinators.
// If this test fails, the change needs a new API version, not a new test value.
//
// The nil capabilitiesDigest case pins the layout of the legacy unversioned /attest endpoint,
// which predates the capabilities binding and must never change.
func TestConstructReportDataGolden(t *testing.T) {
	nonce := make([]byte, 32)
	for i := range nonce {
		nonce[i] = byte(i)
	}
	transitionDigest := make([]byte, 32)
	for i := range transitionDigest {
		transitionDigest[i] = byte(0xa0 + i)
	}
	capabilitiesDigest := make([]byte, 32)
	for i := range capabilitiesDigest {
		capabilitiesDigest[i] = byte(0xc0 + i)
	}
	state := &CoordinatorState{
		RootCA: []byte("root-ca"),
		MeshCA: []byte("mesh-ca"),
	}

	for name, tc := range map[string]struct {
		capabilitiesDigest []byte
		want               string
	}{
		"legacy, without capabilities digest": {
			want: "ec25193fbfa21fb46964de80adca8e7d70222ad41e67c77a696e2f188c02a0f2" +
				"0000000000000000000000000000000000000000000000000000000000000000",
		},
		"v1, with capabilities digest": {
			capabilitiesDigest: capabilitiesDigest,
			want: "8e8d9ec21a9617767c4adb715f712fe246f121211fc76338f0c30824050bbb8e" +
				"0000000000000000000000000000000000000000000000000000000000000000",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := ConstructReportData(nonce, transitionDigest, tc.capabilitiesDigest, state)
			assert.Equal(t, tc.want, hex.EncodeToString(got[:]))
		})
	}
}
