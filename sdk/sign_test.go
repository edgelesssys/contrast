// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignManifestTransition(t *testing.T) {
	key := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[0])
	manifestBytes := []byte("manifest")
	previous := sha256.Sum256([]byte("previous"))

	for name, tc := range map[string]struct {
		previous     []byte
		wantPrevious [history.HashSize]byte
		wantErr      bool
	}{
		"initial manifest": {},
		"update": {
			previous:     previous[:],
			wantPrevious: previous,
		},
		"invalid previous transition hash": {
			previous: []byte{1, 2, 3},
			wantErr:  true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			sig, err := SignManifestTransition(key, manifestBytes, tc.previous)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// This is the scheme of `contrast sign`, as verified by the Coordinator.
			tr := &history.Transition{ManifestHash: sha256.Sum256(manifestBytes), PreviousTransitionHash: tc.wantPrevious}
			transitionHash := tr.Digest()
			signed := sha256.Sum256([]byte(hex.EncodeToString(transitionHash[:])))
			assert.True(t, ecdsa.VerifyASN1(&key.PublicKey, signed[:], sig))
		})
	}
}
