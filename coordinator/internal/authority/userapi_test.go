// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/seedengine"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestManifestSet(t *testing.T) {
	newBaseManifest := func() *manifest.Manifest {
		return &manifest.Manifest{}
	}
	newManifestBytes := func(f func(*manifest.Manifest)) []byte {
		m := newBaseManifest()
		if f != nil {
			f(m)
		}
		b, err := json.Marshal(m)
		require.NoError(t, err)
		return b
	}
	trustedKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[0])
	untrustedKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[1])
	manifestWithTrustedKey, err := manifestWithWorkloadOwnerKey(trustedKey)
	require.NoError(t, err)
	manifestWithoutTrustedKey, err := manifestWithWorkloadOwnerKey(nil)
	require.NoError(t, err)

	testCases := map[string]struct {
		req              *userapi.SetManifestRequest
		workloadOwnerKey *ecdsa.PrivateKey
		wantErr          bool
	}{
		"empty request": {
			req:     &userapi.SetManifestRequest{},
			wantErr: true,
		},
		"manifest without policies": {
			req: &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = nil
				}),
			},
			wantErr: false,
		},
		"request without policies": {
			req: &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = map[manifest.HexString]manifest.PolicyEntry{
						manifest.HexString("a"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
						manifest.HexString("b"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
					}
				}),
			},
			wantErr: true,
		},
		"policy not in manifest": {
			req: &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = map[manifest.HexString]manifest.PolicyEntry{
						manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
						manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
					}
				}),
				Policies: [][]byte{
					[]byte("a"),
					[]byte("c"),
				},
			},
			wantErr: true,
		},
		// TODO(burgerdev): add test for dysfunctional history backend
		"valid manifest": {
			req: &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = map[manifest.HexString]manifest.PolicyEntry{
						manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
						manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
					}
				}),
				Policies: [][]byte{
					[]byte("a"),
					[]byte("b"),
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			reg := prometheus.NewRegistry()
			coordinator := newCoordinatorWithRegistry(reg)
			ctx := rpcContext(tc.workloadOwnerKey)
			resp, err := coordinator.SetManifest(ctx, tc.req)

			if tc.wantErr {
				assert.Error(err)
				requireGauge(t, reg, 0)
				return
			}
			require.NoError(err)
			assert.Equal("system:coordinator:root", parsePEMCertificate(t, resp.RootCA).Subject.CommonName)
			assert.Equal("system:coordinator:intermediate", parsePEMCertificate(t, resp.MeshCA).Subject.CommonName)
			requireGauge(t, reg, 1)
		})
	}

	keyTestCases := map[string]struct {
		workloadOwnerKey *ecdsa.PrivateKey
		wantCode         codes.Code
	}{
		"workload owner key match": {
			workloadOwnerKey: trustedKey,
		},
		"workload owner key mismatch": {
			workloadOwnerKey: untrustedKey,
			wantCode:         codes.PermissionDenied,
		},
		"workload owner key missing": {
			wantCode: codes.PermissionDenied,
		},
	}
	for name, tc := range keyTestCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			reg := prometheus.NewRegistry()
			coordinator := newCoordinatorWithRegistry(reg)
			ctx := rpcContext(tc.workloadOwnerKey)
			m, err := json.Marshal(manifestWithTrustedKey)
			require.NoError(err)
			_, err = coordinator.SetManifest(ctx, &userapi.SetManifestRequest{Manifest: m})
			require.NoError(err)

			req := &userapi.SetManifestRequest{
				Manifest: newManifestBytes(func(m *manifest.Manifest) {
					m.Policies = map[manifest.HexString]manifest.PolicyEntry{
						manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
						manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
					}
				}),
				Policies: [][]byte{
					[]byte("a"),
					[]byte("b"),
				},
			}
			_, err = coordinator.SetManifest(ctx, req)
			require.Equal(tc.wantCode, status.Code(err))
			if tc.wantCode == codes.OK {
				requireGauge(t, reg, 2)
			} else {
				requireGauge(t, reg, 1)
			}
		})
	}

	t.Run("no workload owner key in manifest", func(t *testing.T) {
		require := require.New(t)

		coordinator := newCoordinator()
		ctx := rpcContext(trustedKey)
		m, err := json.Marshal(manifestWithoutTrustedKey)
		require.NoError(err)
		req := &userapi.SetManifestRequest{Manifest: m}
		_, err = coordinator.SetManifest(ctx, req)
		require.NoError(err)
		_, err = coordinator.SetManifest(ctx, req)
		require.Error(err)
		require.Equal(codes.PermissionDenied, status.Code(err))
	})

	t.Run("broken manifest", func(t *testing.T) {
		require := require.New(t)

		coordinator := newCoordinator()
		req := &userapi.SetManifestRequest{Manifest: []byte(`{ "policies": 1 }`)}
		_, err = coordinator.SetManifest(context.Background(), req)
		require.Error(err)
		require.Equal(codes.InvalidArgument, status.Code(err))
	})
}

func TestGetManifests(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	coordinator := newCoordinator()

	ctx := context.Background()
	resp, err := coordinator.GetManifests(ctx, &userapi.GetManifestsRequest{})
	require.Equal(codes.FailedPrecondition, status.Code(err))
	assert.Nil(resp)

	m := &manifest.Manifest{}
	m.Policies = map[manifest.HexString]manifest.PolicyEntry{
		manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
		manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
	}
	manifestBytes, err := json.Marshal(m)
	require.NoError(err)

	req := &userapi.SetManifestRequest{
		Manifest: manifestBytes,
		Policies: [][]byte{
			[]byte("a"),
			[]byte("b"),
		},
	}
	setResp, err := coordinator.SetManifest(ctx, req)
	require.NoError(err)
	assert.NotNil(setResp)

	resp, err = coordinator.GetManifests(ctx, &userapi.GetManifestsRequest{})

	require.NoError(err)
	assert.Equal("system:coordinator:root", parsePEMCertificate(t, resp.RootCA).Subject.CommonName)
	assert.Equal("system:coordinator:intermediate", parsePEMCertificate(t, resp.MeshCA).Subject.CommonName)
	assert.Len(resp.Policies, len(m.Policies))
}

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
			wantCode: codes.OK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			a := newCoordinator()

			seedShareOwnerKey := testkeys.RSA(t)
			seedShareOwnerKeyBytes := manifest.MarshalSeedShareOwnerKey(&seedShareOwnerKey.PublicKey)

			mnfst, _, policies := newManifest(t)
			mnfst.SeedshareOwnerPubKeys = []manifest.HexString{seedShareOwnerKeyBytes}
			manifestBytes, err := json.Marshal(mnfst)
			require.NoError(err)

			req := &userapi.SetManifestRequest{
				Manifest: manifestBytes,
				Policies: policies,
			}
			resp, err := a.SetManifest(context.Background(), req)
			require.NoError(err)
			require.Len(resp.SeedSharesDoc.SeedShares, 1)
			seed, err := manifest.DecryptSeedShare(seedShareOwnerKey, resp.SeedSharesDoc.SeedShares[0])
			require.NoError(err)

			recoverReq := &userapi.RecoverRequest{
				Seed: tc.seed,
				Salt: tc.salt,
			}
			if recoverReq.Seed == nil {
				recoverReq.Seed = seed
			}
			if recoverReq.Salt == nil {
				recoverReq.Salt = resp.SeedSharesDoc.Salt
			}

			// Simulate an updated persistence.
			a.state.Load().stale.Store(true)
			_, err = a.GetManifests(context.Background(), nil)
			require.ErrorContains(err, ErrNeedsRecovery.Error())
			_, err = a.Recover(rpcContext(seedShareOwnerKey), recoverReq)
			require.Equal(tc.wantCode, status.Code(err), "actual error: %v", err)

			// Simulate a restarted Coordinator.
			a = New(a.hist, prometheus.NewRegistry(), slog.Default())
			_, err = a.GetManifests(context.Background(), nil)
			require.ErrorContains(err, ErrNeedsRecovery.Error())
			_, err = a.Recover(rpcContext(seedShareOwnerKey), recoverReq)
			require.Equal(tc.wantCode, status.Code(err), "actual error: %v", err)
		})
	}
}

// TestRecoveryFlow exercises the recovery flow's expected path.
func TestRecoveryFlow(t *testing.T) {
	require := require.New(t)

	// 1. A Coordinator is created from empty state.

	a := newCoordinator()

	// 2. A manifest is set and the returned seed is recorded.
	seedShareOwnerKey := testkeys.RSA(t)
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

	recoverReq := &userapi.RecoverRequest{
		Seed: seed,
		Salt: seedSharesDoc.GetSalt(),
	}

	ctx := rpcContext(seedShareOwnerKey)

	// Recovery on this Coordinator should fail now that a manifest is set.
	_, err = a.Recover(ctx, recoverReq)
	require.ErrorContains(err, ErrAlreadyRecovered.Error())

	// 3. A new Coordinator is created with the existing history.
	// GetManifests and SetManifest are expected to fail.

	a = New(a.hist, prometheus.NewRegistry(), slog.Default())
	_, err = a.SetManifest(context.Background(), req)
	require.ErrorContains(err, ErrNeedsRecovery.Error())

	_, err = a.GetManifests(context.Background(), &userapi.GetManifestsRequest{})
	require.ErrorContains(err, ErrNeedsRecovery.Error())

	// 4. Recovery is called.
	_, err = a.Recover(ctx, recoverReq)
	require.NoError(err)

	// 5. Coordinator should be operational and know about the latest manifest.
	resp, err := a.GetManifests(context.Background(), &userapi.GetManifestsRequest{})
	require.NoError(err)
	require.NotNil(resp)
	require.Len(resp.Manifests, 1)
	require.Equal([][]byte{manifestBytes}, resp.Manifests)

	// Recover on a recovered authority should fail.
	_, err = a.Recover(ctx, recoverReq)
	require.Error(err)

	// Test RecoverWith
	a = New(a.hist, prometheus.NewRegistry(), slog.Default())
	se, err := seedengine.New(seed, seedSharesDoc.GetSalt())
	require.NoError(err)
	latest, err := a.hist.GetLatest(&se.TransactionSigningKey().PublicKey)
	require.NoError(err)
	meshKey := testkeys.ECDSA(t)
	require.NoError(a.RecoverWith(se, latest, meshKey))
	state := a.state.Load()
	require.NotNil(state)
	require.NotNil(state.ca)
	require.Equal(meshKey, state.ca.GetIntermCAPrivKey())

	require.ErrorIs(a.RecoverWith(se, latest, meshKey), ErrAlreadyRecovered)
}

// TestUserAPIConcurrent tests potential synchronization problems between the different
// gRPCs of the server.
func TestUserAPIConcurrent(t *testing.T) {
	newBaseManifest := func() *manifest.Manifest {
		return &manifest.Manifest{}
	}
	newManifestBytes := func(f func(*manifest.Manifest)) []byte {
		m := newBaseManifest()
		if f != nil {
			f(m)
		}
		b, err := json.Marshal(m)
		require.NoError(t, err)
		return b
	}

	fs := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
	store := history.NewAferoStore(&afero.Afero{Fs: fs})
	hist := history.NewWithStore(slog.Default(), store)
	coordinator := New(hist, prometheus.NewRegistry(), slog.Default())

	setReq := &userapi.SetManifestRequest{
		Manifest: newManifestBytes(func(m *manifest.Manifest) {
			m.Policies = map[manifest.HexString]manifest.PolicyEntry{
				manifest.HexString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"): {SANs: []string{"a1", "a2"}, WorkloadSecretID: "a3"},
				manifest.HexString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"): {SANs: []string{"b1", "b2"}, WorkloadSecretID: "b3"},
			}
		}),
		Policies: [][]byte{
			[]byte("a"),
			[]byte("b"),
		},
	}

	ctx := context.Background()
	wg := sync.WaitGroup{}

	set := func() {
		defer wg.Done()
		_, _ = coordinator.SetManifest(ctx, setReq)
	}
	get := func() {
		defer wg.Done()
		_, _ = coordinator.GetManifests(ctx, &userapi.GetManifestsRequest{})
	}

	wg.Add(12)
	go set()
	go set()
	go set()
	go get()
	go get()
	go get()
	go set()
	go set()
	go set()
	go get()
	go get()
	go get()
	wg.Wait()
}

func TestOutOfBandUpdates(t *testing.T) {
	require := require.New(t)
	store := newWatchableStore()
	hist := history.NewWithStore(slog.Default(), store)
	a := newCoordinatorWithWatcher(t, hist)

	// Set an initial manifest.
	seedShareOwnerKey := testkeys.RSA(t)
	seedShareOwnerKeyBytes := manifest.MarshalSeedShareOwnerKey(&seedShareOwnerKey.PublicKey)

	mnfst, _, policies := newManifest(t)
	mnfst.SeedshareOwnerPubKeys = []manifest.HexString{seedShareOwnerKeyBytes}
	manifestBytes, err := json.Marshal(mnfst)
	require.NoError(err)

	req := &userapi.SetManifestRequest{
		Manifest: manifestBytes,
		Policies: policies,
	}
	setManifestResp, err := a.SetManifest(context.Background(), req)
	require.NoError(err)

	// GetManifest should show that the watcher did not mark the state stale.
	getManifestResp, err := a.GetManifests(context.Background(), nil)
	require.NoError(err)
	require.Len(getManifestResp.Manifests, 1)
	require.Equal(manifestBytes, getManifestResp.Manifests[0])

	// Manipulate history directly
	key := a.state.Load().seedEngine.TransactionSigningKey()
	oldLatest, err := hist.GetLatest(&key.PublicKey)
	require.NoError(err)
	transition := &history.Transition{
		ManifestHash:           sha256.Sum256(manifestBytes),
		PreviousTransitionHash: oldLatest.TransitionHash,
	}
	transitionHash, err := hist.SetTransition(transition)
	require.NoError(err)
	nextLatest := &history.LatestTransition{
		TransitionHash: transitionHash,
	}
	require.NoError(hist.SetLatest(oldLatest, nextLatest, key))

	// Wait for the staleness to propagate.
	require.Eventually(func() bool {
		return a.state.Load().stale.Load()
	}, time.Second, 10*time.Millisecond)
	_, err = a.GetManifests(context.Background(), nil)
	require.ErrorContains(err, ErrNeedsRecovery.Error())

	// Recovery should succeed.
	seed, err := manifest.DecryptSeedShare(seedShareOwnerKey, setManifestResp.GetSeedSharesDoc().GetSeedShares()[0])
	require.NoError(err)

	recoverReq := &userapi.RecoverRequest{
		Seed: seed,
		Salt: setManifestResp.GetSeedSharesDoc().GetSalt(),
	}
	_, err = a.Recover(rpcContext(seedShareOwnerKey), recoverReq)
	require.NoError(err)
}

func TestStoreRaces(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	log := slog.Default()

	store := newWatchableStore()
	hist := history.NewWithStore(log, store)
	coordinators := make([]*Authority, 10)
	for i := range coordinators {
		coordinator := newCoordinatorWithWatcher(t, hist)
		coordinators[i] = coordinator
	}

	for i, coordinator := range coordinators {
		_, err := coordinator.GetManifests(ctx, nil)
		assert.ErrorContains(t, err, ErrNoManifest.Error(), "coordinator-%d", i)
	}

	passiveCoordinator := newCoordinatorWithWatcher(t, hist)
	_, err := passiveCoordinator.GetManifests(ctx, nil)
	assert.ErrorContains(t, err, ErrNoManifest.Error())

	seedshareOwnerKey := testkeys.RSA(t)
	workloadOwnerKey := testkeys.ECDSA(t)

	manifestBytes, err := json.Marshal(&manifest.Manifest{
		WorkloadOwnerKeyDigests: []manifest.HexString{manifest.HashWorkloadOwnerKey(&workloadOwnerKey.PublicKey)},
		SeedshareOwnerPubKeys:   []manifest.HexString{manifest.MarshalSeedShareOwnerKey(&seedshareOwnerKey.PublicKey)},
	})
	require.NoError(err)
	req := &userapi.SetManifestRequest{
		Manifest: manifestBytes,
	}

	var seed, salt []byte
	t.Run("parallel initial SetManifest calls", func(t *testing.T) {
		assert := assert.New(t)

		wg := sync.WaitGroup{}
		wg.Add(len(coordinators))

		errs := make(chan error, len(coordinators))
		var seedSharesDoc *userapi.SeedShareDocument
		for _, coordinator := range coordinators {
			go func() {
				resp, err := coordinator.SetManifest(ctx, req)
				errs <- err
				if err == nil {
					seedSharesDoc = resp.GetSeedSharesDoc()
				}
				wg.Done()
			}()
		}
		wg.Wait()
		close(errs)

		nonNil := 0
		for err := range errs {
			if err != nil {
				nonNil++
				// The exact error returned may be different, depending on where in the SetManifest
				// path we detected the concurrent modification (concurrent access to store or
				// staleness update by watcher. Accept that any error is returned for now.
			}
		}
		assert.Equal(len(coordinators)-1, nonNil, "exactly one call should have succeeded")
		assert.NotNil(seedSharesDoc)
		salt = seedSharesDoc.Salt
		assert.Len(seedSharesDoc.SeedShares, 1)
		seed, err = manifest.DecryptSeedShare(seedshareOwnerKey, seedSharesDoc.SeedShares[0])
		assert.NoError(err)
	})

	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		assert := assert.New(t)
		nonNil := 0
		for i, coordinator := range append(coordinators, passiveCoordinator) {
			_, err := coordinator.GetManifests(ctx, nil)
			if err == nil {
				continue
			}
			assert.ErrorContains(err, ErrNeedsRecovery.Error(), "coordinator-%d", i)
			nonNil++
		}
		assert.Equal(len(coordinators), nonNil)
	}, time.Second, 10*time.Millisecond, "coordinators without state must enter recovery mode")

	t.Run("recover coordinators", func(t *testing.T) {
		ctx := rpcContext(seedshareOwnerKey)
		req := &userapi.RecoverRequest{
			Seed: seed,
			Salt: salt,
		}
		for i, coordinator := range append(coordinators, passiveCoordinator) {
			_, err := coordinator.Recover(ctx, req)
			if err != nil {
				assert.ErrorContains(t, err, ErrAlreadyRecovered.Error(), "coordinator-%d", i)
			}
		}
	})

	t.Run("parallel SetManifest calls on initialized coordinators", func(t *testing.T) {
		ctx := rpcContext(workloadOwnerKey)
		assert := assert.New(t)

		wg := sync.WaitGroup{}
		wg.Add(len(coordinators))

		errs := make(chan error, len(coordinators))
		for _, coordinator := range coordinators {
			go func() {
				_, err := coordinator.SetManifest(ctx, req)
				errs <- err
				wg.Done()
			}()
		}
		wg.Wait()
		close(errs)

		nonNil := 0
		for err := range errs {
			if err != nil {
				nonNil++
			}
		}
		assert.Equal(len(coordinators)-1, nonNil, "exactly one call should have succeeded")
	})

	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		assert := assert.New(t)
		nonNil := 0
		for i, coordinator := range append(coordinators, passiveCoordinator) {
			_, err := coordinator.GetManifests(ctx, nil)
			if err == nil {
				continue
			}
			assert.ErrorContains(err, ErrNeedsRecovery.Error(), "coordinator-%d", i)
			nonNil++
		}
		assert.Equal(len(coordinators), nonNil)
	}, time.Second, 300*time.Millisecond, "coordinators with stale state must enter recovery mode")
}

func TestNotificationRaces(t *testing.T) {
	require := require.New(t)

	store := newWatchableStore()
	hist := history.NewWithStore(slog.Default(), store)

	a := newCoordinatorWithWatcher(t, hist)

	// Wait for the first watch call, then swap out the channel so that we control the notifications.
	require.Eventually(func() bool {
		store.mu.Lock()
		defer store.mu.Unlock()
		return len(store.watchers) == 1
	}, 10*time.Millisecond, time.Millisecond)
	store.mu.Lock()
	watchedChs, ok := store.watchers["transitions/latest"]
	require.True(ok)
	require.Len(watchedChs, 1)
	watchedCh := watchedChs[0]
	notifiedCh := make(chan []byte, 1)
	watchedChs[0] = notifiedCh
	store.mu.Unlock()

	// Set two manifests, but don't send notifications yet.
	seedshareOwnerKey := testkeys.RSA(t)
	workloadOwnerKey := testkeys.ECDSA(t)

	manifestBytes, err := json.Marshal(&manifest.Manifest{
		WorkloadOwnerKeyDigests: []manifest.HexString{manifest.HashWorkloadOwnerKey(&workloadOwnerKey.PublicKey)},
		SeedshareOwnerPubKeys:   []manifest.HexString{manifest.MarshalSeedShareOwnerKey(&seedshareOwnerKey.PublicKey)},
	})
	require.NoError(err)
	req := &userapi.SetManifestRequest{
		Manifest: manifestBytes,
	}
	var transitions [][]byte
	for i := range 2 {
		_, err := a.SetManifest(rpcContext(workloadOwnerKey), req)
		require.NoErrorf(err, "SetManifest call %d", i)
		transitions = append(transitions, <-notifiedCh)
	}
	state := a.state.Load()
	require.NotNil(state)
	require.False(state.stale.Load())

	// The manifest is now two steps ahead of the watcher. Verify that it's not marked stale by
	// the notifications arriving late.
	for i, transition := range transitions {
		watchedCh <- transition
		require.Neverf(state.stale.Load, 10*time.Millisecond, time.Millisecond, "notification %d", i)
	}
}

func newCoordinator() *Authority {
	return newCoordinatorWithRegistry(prometheus.NewRegistry())
}

func newCoordinatorWithRegistry(reg *prometheus.Registry) *Authority {
	fs := afero.NewMemMapFs()
	store := history.NewAferoStore(&afero.Afero{Fs: fs})
	hist := history.NewWithStore(slog.Default(), store)
	return New(hist, reg, slog.Default())
}

func newCoordinatorWithWatcher(t *testing.T, hist *history.History) *Authority {
	t.Helper()
	coordinator := New(hist, prometheus.NewRegistry(), slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan struct{})
	go func() {
		_ = coordinator.WatchHistory(ctx)
		close(doneCh)
	}()
	t.Cleanup(func() {
		cancel()
		<-doneCh
	})

	return coordinator
}

func rpcContext(cryptoKey crypto.PrivateKey) context.Context {
	var peerCertificates []*x509.Certificate
	switch key := cryptoKey.(type) {
	case *rsa.PrivateKey:
		if key != nil {
			peerCertificates = append(peerCertificates, &x509.Certificate{PublicKey: key.Public(), PublicKeyAlgorithm: x509.RSA})
		}
	case *ecdsa.PrivateKey:
		if key != nil {
			peerCertificates = append(peerCertificates, &x509.Certificate{PublicKey: key.Public(), PublicKeyAlgorithm: x509.ECDSA})
		}
	default:
		panic(fmt.Sprintf("unsupported key type for rpcContext: %T", cryptoKey))
	}
	return peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{State: tls.ConnectionState{
			PeerCertificates: peerCertificates,
		}},
	})
}

func manifestWithWorkloadOwnerKey(key *ecdsa.PrivateKey) (*manifest.Manifest, error) {
	m := &manifest.Manifest{}
	if key == nil {
		return m, nil
	}
	pubKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	ownerKeyHash := sha256.Sum256(pubKey)
	ownerKeyHex := manifest.NewHexString(ownerKeyHash[:])
	m.WorkloadOwnerKeyDigests = []manifest.HexString{ownerKeyHex}
	return m, nil
}

func parsePEMCertificate(t *testing.T, pemCert []byte) *x509.Certificate {
	t.Helper()

	block, _ := pem.Decode(pemCert)
	require.NotNil(t, block, "no pem-encoded certificate found")

	// Parse the certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	return cert
}

// watchableStore wraps a Store and adds a simple Watch implementation.
type watchableStore struct {
	history.Store
	watchers map[string][]chan []byte
	// mu protects the watchers map to please the race detector.
	mu sync.Mutex
}

func newWatchableStore() *watchableStore {
	fs := afero.NewMemMapFs()
	return &watchableStore{
		Store:    history.NewAferoStore(&afero.Afero{Fs: fs}),
		watchers: make(map[string][]chan []byte),
	}
}

func (fs *watchableStore) Watch(key string) (<-chan []byte, func(), error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	ch := make(chan []byte, 10)
	fs.watchers[key] = append(fs.watchers[key], ch)
	return ch, func() {}, nil
}

func (fs *watchableStore) CompareAndSwap(key string, oldVal, newVal []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	err := fs.Store.CompareAndSwap(key, oldVal, newVal)
	if err != nil {
		return err
	}
	if watchers, ok := fs.watchers[key]; ok {
		for _, watcher := range watchers {
			watcher <- newVal
		}
	}
	return nil
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
