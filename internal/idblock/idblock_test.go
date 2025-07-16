// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package idblock

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"
	"slices"
	"testing"

	"github.com/google/go-sev-guest/abi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoverPublicKey(t *testing.T) {
	empty := [0x48]byte{}
	// A valid R must be an x-coordinate on the curve
	validR := [0x48]byte(append(empty[1:], 0x02))
	// Any S is valid
	validS := [0x48]byte(append(empty[1:], 0x01))

	// Create true ecdsa key
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Sign message
	hash := sha256.Sum256([]byte("message"))
	signedR, signedS, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	testCases := []struct {
		name      string
		r, s      []byte
		pubKey    *ecdsa.PublicKey
		message   string
		expectErr bool
	}{
		{
			name:    "Valid Signature",
			r:       signedR.Bytes(),
			s:       signedS.Bytes(),
			pubKey:  &privateKey.PublicKey,
			message: "message",
		},
		{
			name:    "Valid constant Signature",
			r:       validR[:],
			s:       validS[:],
			message: "message",
		},
		{
			name:      "Invalid Signature",
			r:         []byte{0x00},
			s:         []byte{0x00},
			message:   "message",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			curve := elliptic.P384()
			bigR := new(big.Int).SetBytes(tc.r)
			bigS := new(big.Int).SetBytes(tc.s)
			hash := sha256.Sum256([]byte(tc.message))
			z := new(big.Int).SetBytes(hash[:])

			publicKeys, err := recoverPublicKey(curve, bigR, bigS, z)
			if tc.expectErr {
				assert.Error(err)
				return
			}

			require.NoError(err)

			require.Len(publicKeys, 2)

			for _, pubKey := range publicKeys {
				assert.True(ecdsa.Verify(&pubKey, hash[:], bigR, bigS))
			}

			if tc.pubKey != nil {
				assert.True(slices.ContainsFunc(publicKeys, func(pk ecdsa.PublicKey) bool {
					return pk.X.Cmp(tc.pubKey.X) == 0 && pk.Y.Cmp(tc.pubKey.Y) == 0
				}))
			}
		})
	}
}

func TestIDBlocksFromLaunchDigest(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	launchDigest := [48]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F, 0x30,
	}

	idblk, idAuth, err := IDBlocksFromLaunchDigest(launchDigest, abi.SnpPolicy{})
	require.NoError(err)

	// Check some specific values in the idBlockBytes
	assert.Equal(launchDigest, idblk.LD)
	assert.Equal(uint32(0x1), idblk.Version)

	// Check some specific values in the idAuthBytes
	assert.Equal(uint32(0x1), idAuth.IDKeyAlgo)
	assert.Equal(uint32(0x0), idAuth.AuthKeyAlgo)
	assert.Equal(uint32(0x2), idAuth.IDKey.CurveID) // Curve ID of the public key
}
