// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apiv1

import (
	"bytes"
	"crypto/sha256"
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

// TestConstructAttestationReportDataGolden pins the byte layout of the report data digest of the versioned endpoint.
//
// The same caveat as for [TestConstructReportDataGolden] applies: a failure means the change needs a new API version.
func TestConstructAttestationReportDataGolden(t *testing.T) {
	nonce := make([]byte, NonceSize)
	for i := range nonce {
		nonce[i] = byte(i)
	}
	transitionDigest := make([]byte, DigestSize)
	for i := range transitionDigest {
		transitionDigest[i] = byte(0xa0 + i)
	}
	capabilitiesDigest := make([]byte, DigestSize)
	for i := range capabilitiesDigest {
		capabilitiesDigest[i] = byte(0xc0 + i)
	}

	for name, tc := range map[string]struct {
		transitionDigest []byte
		resp             *AttestationResponse
		// wantPreimage spells out the layout independently of the implementation.
		wantPreimage [][]byte
		want         string
	}{
		"active": {
			transitionDigest: transitionDigest,
			resp: &AttestationResponse{
				Status:                    StatusActive,
				HPKEPubKey:                []byte("hpke-pub-key"),
				CapabilitiesDigest:        capabilitiesDigest,
				LatestTransitionSignature: []byte("transition-signature"),
				CoordinatorState:          CoordinatorState{RootCA: []byte("root-ca"), MeshCA: []byte("mesh-ca")},
			},
			wantPreimage: [][]byte{
				sum("contrast attest v1"), nonce,
				{0x01},
				transitionDigest, sum("transition-signature"),
				sum("root-ca"), sum("mesh-ca"), sum("hpke-pub-key"), capabilitiesDigest,
			},
			want: "a029eee550262d615f1b2c7667ab3ee16e4cbdf325f755780852e4720419c5d2" +
				"0000000000000000000000000000000000000000000000000000000000000000",
		},
		"stale": {
			resp: &AttestationResponse{
				Status:             StatusStale,
				HPKEPubKey:         []byte("hpke-pub-key"),
				CapabilitiesDigest: capabilitiesDigest,
			},
			wantPreimage: [][]byte{
				sum("contrast attest v1"), nonce,
				{0x02},
				make([]byte, 32), sum(""),
				sum(""), sum(""), sum("hpke-pub-key"), capabilitiesDigest,
			},
			want: "dd0ce156f8c7a13a2c6117482ae20b1528867506b5bd6dfcfebaf412edf5089b" +
				"0000000000000000000000000000000000000000000000000000000000000000",
		},
		"uninitialized": {
			resp: &AttestationResponse{
				Status:             StatusUninitialized,
				HPKEPubKey:         []byte("hpke-pub-key"),
				CapabilitiesDigest: capabilitiesDigest,
			},
			wantPreimage: [][]byte{
				sum("contrast attest v1"), nonce,
				{0x00},
				make([]byte, 32), sum(""),
				sum(""), sum(""), sum("hpke-pub-key"), capabilitiesDigest,
			},
			want: "888c6a914cf59659e896581b0c447e73f76688ff51c88a43948c0d660e99566c" +
				"0000000000000000000000000000000000000000000000000000000000000000",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			got, err := ConstructAttestationReportData(nonce, tc.transitionDigest, tc.resp)
			assert.NoError(err)
			assert.Equal(tc.want, hex.EncodeToString(got[:]))

			wantDigest := sha256.Sum256(bytes.Join(tc.wantPreimage, nil))
			assert.Equal(wantDigest[:], got[:sha256.Size])
		})
	}
}

func TestConstructAttestationReportDataRejectsBadInput(t *testing.T) {
	valid := func() *AttestationResponse {
		return &AttestationResponse{Status: StatusActive, CapabilitiesDigest: make([]byte, DigestSize)}
	}

	for name, tc := range map[string]struct {
		nonce            []byte
		transitionDigest []byte
		mutate           func(*AttestationResponse)
	}{
		"short nonce": {
			nonce:            make([]byte, NonceSize-1),
			transitionDigest: make([]byte, DigestSize),
		},
		"long transition digest": {
			nonce:            make([]byte, NonceSize),
			transitionDigest: make([]byte, DigestSize+1),
		},
		"active without transition digest": {
			nonce: make([]byte, NonceSize),
		},
		"unknown status": {
			nonce:            make([]byte, NonceSize),
			transitionDigest: make([]byte, DigestSize),
			mutate:           func(r *AttestationResponse) { r.Status = "bogus" },
		},
		"no status": {
			nonce:            make([]byte, NonceSize),
			transitionDigest: make([]byte, DigestSize),
			mutate:           func(r *AttestationResponse) { r.Status = "" },
		},
		"no capabilities digest": {
			nonce:            make([]byte, NonceSize),
			transitionDigest: make([]byte, DigestSize),
			mutate:           func(r *AttestationResponse) { r.CapabilitiesDigest = nil },
		},
	} {
		t.Run(name, func(t *testing.T) {
			resp := valid()
			if tc.mutate != nil {
				tc.mutate(resp)
			}
			_, err := ConstructAttestationReportData(tc.nonce, tc.transitionDigest, resp)
			assert.Error(t, err)
		})
	}
}

func sum(s string) []byte {
	digest := sha256.Sum256([]byte(s))
	return digest[:]
}
