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
// Only the layout of the unversioned /attest endpoint is pinned, which is released and must never change.
func TestConstructReportDataGolden(t *testing.T) {
	nonce := make([]byte, 32)
	for i := range nonce {
		nonce[i] = byte(i)
	}
	transitionDigest := make([]byte, 32)
	for i := range transitionDigest {
		transitionDigest[i] = byte(0xa0 + i)
	}
	state := &CoordinatorState{
		RootCA: []byte("root-ca"),
		MeshCA: []byte("mesh-ca"),
	}

	for name, tc := range map[string]struct {
		got  [ReportDataSize]byte
		want string
	}{
		"legacy, without capabilities digest": {
			got: ConstructReportData(nonce, transitionDigest, state),
			want: "ec25193fbfa21fb46964de80adca8e7d70222ad41e67c77a696e2f188c02a0f2" +
				"0000000000000000000000000000000000000000000000000000000000000000",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, hex.EncodeToString(tc.got[:]))
		})
	}
}
