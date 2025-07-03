// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package userapi

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
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/coordinator/internal/history"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/google/go-sev-guest/abi"
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

func TestSetManifest(t *testing.T) {
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
			ctx := rpcContext(t.Context(), tc.workloadOwnerKey)
			resp, err := coordinator.SetManifest(ctx, tc.req)

			if tc.wantErr {
				assert.Error(err)
				return
			}
			require.NoError(err)
			assert.Equal("system:coordinator:root", parsePEMCertificate(t, resp.RootCA).Subject.CommonName)
			assert.Equal("system:coordinator:intermediate", parsePEMCertificate(t, resp.MeshCA).Subject.CommonName)
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
			ctx := rpcContext(t.Context(), tc.workloadOwnerKey)
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
		})
	}

	t.Run("no workload owner key in manifest", func(t *testing.T) {
		require := require.New(t)

		coordinator := newCoordinator()
		ctx := rpcContext(t.Context(), trustedKey)
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
		_, err = coordinator.SetManifest(t.Context(), req)
		require.Error(err)
		require.Equal(codes.InvalidArgument, status.Code(err))
	})
}

func TestGetManifests(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	coordinator := newCoordinator()

	ctx := t.Context()
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
		name        string
		seed        *[]byte
		salt        *[]byte
		force       bool
		peers       []string
		peersErr    error
		wantCode    codes.Code
		wantMessage string
	}{
		{
			name:        "empty seed",
			seed:        toPtr[[]byte](nil),
			wantCode:    codes.InvalidArgument,
			wantMessage: "seed must be",
		},
		{
			name:        "empty salt",
			salt:        toPtr[[]byte](nil),
			wantCode:    codes.InvalidArgument,
			wantMessage: "salt must be",
		},
		{
			name:        "short seed",
			seed:        toPtr(seed[:16]),
			wantCode:    codes.InvalidArgument,
			wantMessage: "seed must be",
		},
		{
			name:        "short salt",
			salt:        toPtr(salt[:16]),
			wantCode:    codes.InvalidArgument,
			wantMessage: "salt must be",
		},
		{
			name:        "peer available",
			peers:       []string{"192.0.2.2"},
			wantCode:    codes.FailedPrecondition,
			wantMessage: "peers are available",
		},
		{
			name:        "peer discovery broken",
			peersErr:    assert.AnError,
			wantCode:    codes.Internal,
			wantMessage: assert.AnError.Error(),
		},
		{
			name:     "peer available but forced",
			force:    true,
			peers:    []string{"192.0.2.2"},
			wantCode: codes.OK,
		},
		{
			name:     "peer discovery broken but forced",
			force:    true,
			peersErr: assert.AnError,
			wantCode: codes.OK,
		},
		{
			name:     "normal values",
			wantCode: codes.OK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			logger := slog.Default()
			fs := afero.NewMemMapFs()
			store := history.NewAferoStore(&afero.Afero{Fs: fs})
			hist := history.NewWithStore(slog.Default(), store)
			auth := stateguard.New(hist, prometheus.NewRegistry(), logger)
			discovery := &stubDiscovery{
				peers: tc.peers,
				err:   tc.peersErr,
			}
			a := New(logger, auth, discovery)

			manifestBytes, policies := newManifestWithSeedshareOwner(t)

			req := &userapi.SetManifestRequest{
				Manifest: manifestBytes,
				Policies: policies,
			}
			resp, err := a.SetManifest(t.Context(), req)
			require.NoError(err)
			require.Len(resp.SeedSharesDoc.SeedShares, 1)
			seedShareOwnerKey := testkeys.RSA(t)
			seed, err := manifest.DecryptSeedShare(seedShareOwnerKey, resp.SeedSharesDoc.SeedShares[0])
			require.NoError(err)

			recoverReq := &userapi.RecoverRequest{
				Seed:  seed,
				Salt:  resp.SeedSharesDoc.Salt,
				Force: tc.force,
			}
			// Override with test case data, if present.
			if tc.seed != nil {
				recoverReq.Seed = *tc.seed
			}
			if tc.salt != nil {
				recoverReq.Salt = *tc.salt
			}

			// Simulate a restarted Coordinator.
			a.guard = stateguard.New(hist, prometheus.NewRegistry(), slog.Default())
			_, err = a.GetManifests(t.Context(), nil)
			require.ErrorContains(err, ErrNeedsRecovery.Error())
			_, err = a.Recover(rpcContext(t.Context(), seedShareOwnerKey), recoverReq)
			require.Equal(tc.wantCode, status.Code(err), "actual error: %v", err)
			if tc.wantMessage != "" {
				require.ErrorContains(err, tc.wantMessage)
			}
		})
	}
}

// TestRecoveryFlow exercises the recovery flow's expected path.
func TestRecoveryFlow(t *testing.T) {
	require := require.New(t)

	// 1. A Coordinator is created from empty state.

	logger := slog.Default()
	fs := afero.NewMemMapFs()
	store := history.NewAferoStore(&afero.Afero{Fs: fs})
	hist := history.NewWithStore(slog.Default(), store)
	auth := stateguard.New(hist, prometheus.NewRegistry(), logger)
	a := New(logger, auth, &stubDiscovery{})

	// 2. A manifest is set and the returned seed is recorded.
	manifestBytes, policies := newManifestWithSeedshareOwner(t)

	req := &userapi.SetManifestRequest{
		Manifest: manifestBytes,
		Policies: policies,
	}
	resp1, err := a.SetManifest(t.Context(), req)
	require.NoError(err)
	require.NotNil(resp1)
	seedSharesDoc := resp1.GetSeedSharesDoc()
	require.NotNil(seedSharesDoc)
	seedShares := seedSharesDoc.GetSeedShares()
	require.Len(seedShares, 1)

	seedShareOwnerKey := testkeys.RSA(t)
	seed, err := manifest.DecryptSeedShare(seedShareOwnerKey, seedShares[0])
	require.NoError(err)

	recoverReq := &userapi.RecoverRequest{
		Seed: seed,
		Salt: seedSharesDoc.GetSalt(),
	}

	ctx := rpcContext(t.Context(), seedShareOwnerKey)

	// Recovery on this Coordinator should fail now that a manifest is set.
	_, err = a.Recover(ctx, recoverReq)
	require.ErrorContains(err, ErrAlreadyRecovered.Error())

	// 3. A new Coordinator is created with the existing history.
	// GetManifests and SetManifest are expected to fail.

	a.guard = stateguard.New(hist, prometheus.NewRegistry(), slog.Default())
	_, err = a.SetManifest(t.Context(), req)
	require.ErrorContains(err, ErrNeedsRecovery.Error())

	_, err = a.GetManifests(t.Context(), &userapi.GetManifestsRequest{})
	require.ErrorContains(err, ErrNeedsRecovery.Error())

	// 4. Recovery is called.
	_, err = a.Recover(ctx, recoverReq)
	require.NoError(err)

	// 5. Coordinator should be operational and know about the latest manifest.
	resp, err := a.GetManifests(t.Context(), &userapi.GetManifestsRequest{})
	require.NoError(err)
	require.NotNil(resp)
	require.Len(resp.Manifests, 1)
	require.Equal([][]byte{manifestBytes}, resp.Manifests)

	// Recover on a recovered Guard should fail.
	_, err = a.Recover(ctx, recoverReq)
	require.Error(err)
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

	logger := slog.Default()
	fs := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
	store := history.NewAferoStore(&afero.Afero{Fs: fs})
	hist := history.NewWithStore(slog.Default(), store)
	auth := stateguard.New(hist, prometheus.NewRegistry(), logger)
	coordinator := New(logger, auth, &stubDiscovery{})

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

	ctx := t.Context()
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
	manifestBytes, policies := newManifestWithSeedshareOwner(t)

	req := &userapi.SetManifestRequest{
		Manifest: manifestBytes,
		Policies: policies,
	}
	setManifestResp, err := a.SetManifest(t.Context(), req)
	require.NoError(err)

	// GetManifest should show that the watcher did not mark the state stale.
	getManifestResp, err := a.GetManifests(t.Context(), nil)
	require.NoError(err)
	require.Len(getManifestResp.Manifests, 1)
	require.Equal(manifestBytes, getManifestResp.Manifests[0])

	// Manipulate history directly
	state, err := a.guard.GetState(t.Context())
	require.NoError(err)
	key := state.SeedEngine().TransactionSigningKey()
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
		_, err := a.guard.GetState(t.Context())
		return errors.Is(err, stateguard.ErrStaleState)
	}, time.Second, 10*time.Millisecond)
	_, err = a.GetManifests(t.Context(), nil)
	require.ErrorContains(err, ErrNeedsRecovery.Error())

	// Recovery should succeed.
	seedShareOwnerKey := testkeys.RSA(t)
	seed, err := manifest.DecryptSeedShare(seedShareOwnerKey, setManifestResp.GetSeedSharesDoc().GetSeedShares()[0])
	require.NoError(err)

	recoverReq := &userapi.RecoverRequest{
		Seed: seed,
		Salt: setManifestResp.GetSeedSharesDoc().GetSalt(),
	}
	_, err = a.Recover(rpcContext(t.Context(), seedShareOwnerKey), recoverReq)
	require.NoError(err)
}

func TestStoreRaces(t *testing.T) {
	require := require.New(t)
	ctx := t.Context()
	log := slog.Default()

	store := newWatchableStore()
	hist := history.NewWithStore(log, store)
	coordinators := make([]*Server, 10)
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
		ctx := rpcContext(t.Context(), seedshareOwnerKey)
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
		ctx := rpcContext(t.Context(), workloadOwnerKey)
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
		_, err := a.SetManifest(rpcContext(t.Context(), workloadOwnerKey), req)
		require.NoErrorf(err, "SetManifest call %d", i)
		transitions = append(transitions, <-notifiedCh)
	}
	state, err := a.guard.GetState(t.Context())
	require.NoError(err)
	require.NotNil(state)

	// The manifest is now two steps ahead of the watcher. Verify that it's not marked stale by
	// the notifications arriving late.
	for i, transition := range transitions {
		watchedCh <- transition
		require.Neverf(func() bool {
			_, err := a.guard.GetState(t.Context())
			return err != nil
		}, 10*time.Millisecond, time.Millisecond, "notification %d", i)
	}
}

func newCoordinator() *Server {
	return newCoordinatorWithRegistry(prometheus.NewRegistry())
}

func newCoordinatorWithRegistry(reg *prometheus.Registry) *Server {
	logger := slog.Default()
	fs := afero.NewMemMapFs()
	store := history.NewAferoStore(&afero.Afero{Fs: fs})
	hist := history.NewWithStore(slog.Default(), store)
	auth := stateguard.New(hist, reg, logger)
	return New(logger, auth, &stubDiscovery{})
}

func newCoordinatorWithWatcher(t *testing.T, hist *history.History) *Server {
	t.Helper()
	logger := slog.Default()
	auth := stateguard.New(hist, prometheus.NewRegistry(), logger)
	coordinator := New(logger, auth, &stubDiscovery{})

	ctx, cancel := context.WithCancel(t.Context())
	doneCh := make(chan struct{})
	go func() {
		_ = auth.WatchHistory(ctx)
		close(doneCh)
	}()
	t.Cleanup(func() {
		cancel()
		<-doneCh
	})

	return coordinator
}

func newManifestWithSeedshareOwner(t *testing.T) ([]byte, [][]byte) {
	t.Helper()
	policy := []byte("=== SOME REGO HERE ===")
	policyHash := sha256.Sum256(policy)
	policyHashHex := manifest.NewHexString(policyHash[:])

	mnfst := &manifest.Manifest{}
	mnfst.Policies = map[manifest.HexString]manifest.PolicyEntry{
		policyHashHex: {
			SANs:             []string{"test"},
			WorkloadSecretID: "test2",
			Role:             manifest.RoleCoordinator,
		},
	}
	svn0 := manifest.SVN(0)
	measurement := [48]byte{}
	mnfst.ReferenceValues.SNP = []manifest.SNPReferenceValues{{
		ProductName: "Milan",
		MinimumTCB: manifest.SNPTCB{
			BootloaderVersion: &svn0,
			TEEVersion:        &svn0,
			SNPVersion:        &svn0,
			MicrocodeVersion:  &svn0,
		},
		TrustedMeasurement: manifest.NewHexString(measurement[:]),
		GuestPolicy: abi.SnpPolicy{
			SMT: true,
		},
	}}
	workloadOwnerKey := testkeys.ECDSA(t)
	workloadOwnerKeyDigest := manifest.HashWorkloadOwnerKey(&workloadOwnerKey.PublicKey)
	mnfst.WorkloadOwnerKeyDigests = []manifest.HexString{workloadOwnerKeyDigest}
	seedShareOwnerKey := testkeys.RSA(t)
	seedShareOwnerKeyBytes := manifest.MarshalSeedShareOwnerKey(&seedShareOwnerKey.PublicKey)
	mnfst.SeedshareOwnerPubKeys = []manifest.HexString{seedShareOwnerKeyBytes}
	mnfstBytes, err := json.Marshal(mnfst)
	require.NoError(t, err)
	return mnfstBytes, [][]byte{policy}
}

func rpcContext(ctx context.Context, cryptoKey crypto.PrivateKey) context.Context {
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
	return peer.NewContext(ctx, &peer.Peer{
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

type stubDiscovery struct {
	peers []string
	err   error
}

func (d *stubDiscovery) GetPeers(ctx context.Context) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return d.peers, d.err
	}
}

func toPtr[A any](a A) *A {
	return &a
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
