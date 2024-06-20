// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/recoveryapi"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecovery(t *testing.T) {
	var seed [32]byte
	var salt [32]byte
	testCases := []struct {
		name     string
		seed     []byte
		salt     []byte
		wantCode codes.Code
	}{
		{
			name:     "empty seed",
			salt:     salt[:],
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "empty salt",
			seed:     seed[:],
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "short seed",
			seed:     seed[:16],
			salt:     salt[:],
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "short salt",
			seed:     seed[:],
			salt:     salt[:16],
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "normal values",
			seed:     seed[:],
			salt:     salt[:],
			wantCode: codes.OK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			a := newCoordinator()
			_, err := a.Recover(context.Background(), &recoveryapi.RecoverRequest{
				Seed: tc.seed,
				Salt: tc.salt,
			})

			require.Equal(tc.wantCode, status.Code(err))
		})
	}
}

// TestRecoveryFlow exercises the recovery flow's expected path.
func TestRecoveryFlow(t *testing.T) {
	require := require.New(t)

	// 1. A Coordinator is created from empty state.

	a := newCoordinator()

	// 2. A manifest is set and the returned seed is recorded.
	seedShareOwnerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(err)
	seedShareOwnerKeyBytes := manifest.MarshalSeedShareOwnerKey(&seedShareOwnerKey.PublicKey)

	mnfst, _, policies := newManifest(t)
	mnfst.SeedshareOwnerPubKeys = []manifest.HexString{seedShareOwnerKeyBytes}
	manifestBytes, err := json.Marshal(mnfst)
	require.NoError(err)

	req := &userapi.SetManifestRequest{
		Manifest: manifestBytes,
		Policies: policies,
	}
	resp1, err := a.SetManifest(context.Background(), req)
	require.NoError(err)
	require.NotNil(resp1)
	seedSharesDoc := resp1.GetSeedSharesDoc()
	require.NotNil(seedSharesDoc)
	seedShares := seedSharesDoc.GetSeedShares()
	require.Len(seedShares, 1)

	seed, err := manifest.DecryptSeedShare(seedShareOwnerKey, seedShares[0])
	require.NoError(err)

	recoverReq := &recoveryapi.RecoverRequest{
		Seed: seed,
		Salt: seedSharesDoc.GetSalt(),
	}

	// Recovery on this Coordinator should fail now that a manifest is set.
	_, err = a.Recover(context.Background(), recoverReq)
	require.ErrorContains(err, ErrAlreadyRecovered.Error())

	// 3. A new Coordinator is created with the existing history.
	// GetManifests and SetManifest are expected to fail.

	a = New(a.hist, prometheus.NewRegistry(), slog.Default())
	_, err = a.SetManifest(context.Background(), req)
	require.ErrorContains(err, ErrNeedsRecovery.Error())

	_, err = a.GetManifests(context.Background(), &userapi.GetManifestsRequest{})
	require.ErrorContains(err, ErrNeedsRecovery.Error())

	// 4. Recovery is called.
	_, err = a.Recover(context.Background(), recoverReq)
	require.NoError(err)

	// 5. Coordinator should be operational and know about the latest manifest.
	resp, err := a.GetManifests(context.Background(), &userapi.GetManifestsRequest{})
	require.NoError(err)
	require.NotNil(resp)
	require.Len(resp.Manifests, 1)
	require.Equal([][]byte{manifestBytes}, resp.Manifests)

	// Recover on a recovered authority should fail.
	_, err = a.Recover(context.Background(), recoverReq)
	require.Error(err)
}
