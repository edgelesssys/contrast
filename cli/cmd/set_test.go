// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetCmdExperimentalHTTPFlag ensures the HTTP API is opt-in, so that existing invocations keep using gRPC.
func TestSetCmdExperimentalHTTPFlag(t *testing.T) {
	require := require.New(t)

	flag := NewSetCmd().Flags().Lookup("experimental-http")
	require.NotNil(flag)
	require.Equal("false", flag.DefValue)
}

func TestHTTPAPIBaseURL(t *testing.T) {
	for name, tc := range map[string]struct {
		coordinator string
		want        string
		wantErr     bool
	}{
		"host and port": {
			coordinator: "coordinator.default:1313",
			want:        "http://coordinator.default:1314",
		},
		"host without port": {
			coordinator: "coordinator.default",
			want:        "http://coordinator.default:1314",
		},
		"IPv4": {
			coordinator: "127.0.0.1:1313",
			want:        "http://127.0.0.1:1314",
		},
		"IPv6": {
			coordinator: "[::1]:1313",
			want:        "http://[::1]:1314",
		},
		"empty": {
			coordinator: "",
			wantErr:     true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := httpAPIBaseURL(tc.coordinator)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestSignTransition ensures the CLI produces the signature the Coordinator expects.
func TestSignTransition(t *testing.T) {
	require := require.New(t)

	key := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0])
	manifestBytes := []byte(`{"foo":"bar"}`)
	previousTransitionHash := make([]byte, history.HashSize)
	for i := range previousTransitionHash {
		previousTransitionHash[i] = byte(i)
	}

	sig, err := signTransition(manifestBytes, previousTransitionHash, key)
	require.NoError(err)

	// Recompute the digest the way the Coordinator does when validating.
	tr := &history.Transition{
		ManifestHash:           history.Digest(manifestBytes),
		PreviousTransitionHash: [history.HashSize]byte(previousTransitionHash),
	}
	transitionHash := tr.Digest()
	signingHash := sha256.Sum256(hex.AppendEncode(nil, transitionHash[:]))

	require.True(ecdsa.VerifyASN1(&key.PublicKey, signingHash[:], sig))
}

func TestSignTransitionRejectsBadHashLength(t *testing.T) {
	key := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0])
	_, err := signTransition([]byte("{}"), []byte("too short"), key)
	require.ErrorContains(t, err, "invalid latest transition hash byte length")
}

// TestSetManifestResponseFromAPI ensures the HTTP response is converted to the gRPC shape, so
// that the files written to disk don't depend on the transport.
func TestSetManifestResponseFromAPI(t *testing.T) {
	t.Run("with seed shares", func(t *testing.T) {
		require := require.New(t)
		assert := assert.New(t)

		got := setManifestResponseFromAPI(&apitypes.SetManifestResponse{
			RootCA: []byte("root-ca"),
			MeshCA: []byte("mesh-ca"),
			SeedSharesDoc: &apitypes.SeedShareDocument{
				Salt: []byte("salt"),
				SeedShares: []apitypes.SeedShare{{
					PublicKey:     "public-key",
					EncryptedSeed: []byte("encrypted-seed"),
				}},
			},
		})

		assert.Equal([]byte("root-ca"), got.RootCA)
		assert.Equal([]byte("mesh-ca"), got.MeshCA)
		require.NotNil(got.SeedSharesDoc)
		assert.Equal([]byte("salt"), got.SeedSharesDoc.Salt)
		require.Len(got.SeedSharesDoc.SeedShares, 1)
		assert.Equal("public-key", got.SeedSharesDoc.SeedShares[0].PublicKey)
		assert.Equal([]byte("encrypted-seed"), got.SeedSharesDoc.SeedShares[0].EncryptedSeed)

		// The on-disk format must stay the one `contrast recover` parses.
		marshaled, err := json.Marshal(got.SeedSharesDoc)
		require.NoError(err)
		var roundTripped userapi.SeedShareDocument
		require.NoError(json.Unmarshal(marshaled, &roundTripped))
		assert.Equal([]byte("salt"), roundTripped.Salt)
		require.Len(roundTripped.SeedShares, 1)
		assert.Equal("public-key", roundTripped.SeedShares[0].PublicKey)
	})

	t.Run("without seed shares", func(t *testing.T) {
		got := setManifestResponseFromAPI(&apitypes.SetManifestResponse{
			RootCA: []byte("root-ca"),
			MeshCA: []byte("mesh-ca"),
		})
		require.Nil(t, got.SeedSharesDoc)
	})
}
