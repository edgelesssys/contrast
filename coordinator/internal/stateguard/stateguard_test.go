// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package stateguard

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/coordinator/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/seedengine"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/google/go-sev-guest/abi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testingclock "k8s.io/utils/clock/testing"
)

const (
	manifestGenerationExpected = `
# HELP contrast_coordinator_manifest_generation Current manifest generation.
# TYPE contrast_coordinator_manifest_generation gauge
contrast_coordinator_manifest_generation %d
`
)

func TestUpdateState(t *testing.T) {
	ctx := t.Context()
	assert := assert.New(t)
	require := require.New(t)
	g, _ := newTestGuard(t)

	// A new Guard is empty.
	emptyState, err := g.GetState(ctx)
	require.Nil(emptyState)
	require.ErrorIs(err, ErrNoState)

	mnfst, manifestBytes, policies := newManifest(t)

	se := newSeedEngine(t)
	updateState, err := g.UpdateState(ctx, emptyState, se, manifestBytes, policies)
	require.NoError(err)
	require.NotNil(updateState)

	// GetState now returns the same state as UpdateState.
	getState, err := g.GetState(ctx)
	require.NoError(err)
	require.NotNil(getState)
	require.Equal(getState, updateState)

	// The returned state contains the values passed to UpdateState.
	assert.Equal(mnfst, getState.Manifest())
	assert.Equal(manifestBytes, getState.ManifestBytes())
	assert.Same(se, getState.SeedEngine())

	// The state has a CA with valid certs.
	ca := getState.CA()
	require.NotNil(ca)
	for name, pemBytes := range map[string][]byte{
		"root CA":         ca.GetRootCACert(),
		"intermediate CA": ca.GetIntermCACert(),
		"mesh CA":         ca.GetMeshCACert(),
	} {
		pool := x509.NewCertPool()
		assert.True(pool.AppendCertsFromPEM(pemBytes), name)
		block, rest := pem.Decode(pemBytes)
		assert.NotEmpty(block.Bytes, name)
		assert.Empty(rest, name)
	}
}

// TestTestConcurrentStateUpdate tests that parallel changes to the internal state pointer don't affect the
// outcome of UpdateState calls.
func TestTestConcurrentStateUpdate(t *testing.T) {
	ctx := t.Context()
	assert := assert.New(t)
	require := require.New(t)
	g, _ := newTestGuard(t)

	// A new Guard is empty.
	emptyState, err := g.GetState(ctx)
	require.Nil(emptyState)
	require.ErrorIs(err, ErrNoState)

	se := newSeedEngine(t)
	_, manifestBytes, policies := newManifest(t)

	// Simulate a concurrent state update.
	concurrentlyUpdatedState := &State{}
	g.state.Store(concurrentlyUpdatedState)

	updateState, err := g.UpdateState(ctx, emptyState, se, manifestBytes, policies)
	require.NoError(err)
	require.NotNil(updateState)
	assert.NotSame(concurrentlyUpdatedState, updateState, "UpdateState must return the state corresponding to its inputs")

	getState, err := g.GetState(ctx)
	require.NoError(err)
	require.Same(concurrentlyUpdatedState, getState, "UpdateState must not override a relatively newer state")
}

func TestGetHistory(t *testing.T) {
	ctx := t.Context()
	assert := assert.New(t)
	require := require.New(t)
	g, _ := newTestGuard(t)

	mnfst, _, policies := newManifest(t)
	se := newSeedEngine(t)

	// Set a number of slightly different manifests to establish a history.
	numManifests := 20
	var state *State
	for i := range numManifests {
		nextPolicy := []byte{byte(i)}
		nextPolicyHash := sha256.Sum256(nextPolicy)
		mnfst.Policies[manifest.NewHexString(nextPolicyHash[:])] = manifest.PolicyEntry{}
		policies = append(policies, nextPolicy)
		manifestBytes, err := json.Marshal(mnfst)
		require.NoError(err)
		nextState, err := g.UpdateState(ctx, state, se, manifestBytes, policies)
		require.NoError(err)
		state = nextState
	}

	// Verify manifest history.
	manifests, policiesByHash, err := g.GetHistory(ctx)
	require.NoError(err)
	assert.Len(policiesByHash, numManifests+1) // 1 additional policy comes from newManifest
	require.Len(manifests, numManifests)
	for i := range numManifests {
		var unmarshaled manifest.Manifest
		require.NoError(json.Unmarshal(manifests[i], &unmarshaled))
		assert.Len(unmarshaled.Policies, i+2) // 1 policy added per iteration + 1 from newManifest
		for hash := range unmarshaled.Policies {
			assert.Contains(policiesByHash, hash)
		}
	}
}

func TestResetState(t *testing.T) {
	ctx := t.Context()
	require := require.New(t)
	g, _ := newTestGuard(t)
	se := newSeedEngine(t)
	_, manifestBytes, policies := newManifest(t)

	authz := &stubAuthorizer{
		se: se,
		pk: testkeys.ECDSA(t),
	}

	// Reset without a persisted state should fail.
	state, err := g.ResetState(ctx, nil, authz)
	require.Error(err)
	require.Nil(state)

	// Initialize the state.
	state, err = g.UpdateState(ctx, nil, se, manifestBytes, policies)
	require.NoError(err)
	require.NotNil(state)

	// Reset with stale state should fail.
	_, err = g.ResetState(ctx, nil, authz)
	require.ErrorIs(err, ErrConcurrentUpdate)

	// Reset with current state should pass.
	state, err = g.ResetState(ctx, state, authz)
	require.NoError(err)
	require.NotNil(state)

	// Unauthorized state reset should fail.
	unauthz := &stubAuthorizer{err: assert.AnError}
	_, err = g.ResetState(ctx, nil, unauthz)
	require.ErrorIs(err, assert.AnError)
}

func TestConcurrentUpdateState(t *testing.T) {
	ctx := t.Context()
	assert := assert.New(t)

	store := &storeWithSync{
		Store: history.NewAferoStore(&afero.Afero{Fs: afero.NewMemMapFs()}),
	}
	hist := history.NewWithStore(slog.Default(), store)
	guard := New(hist, prometheus.NewRegistry(), slog.Default())

	numWorkers := 20

	se := newSeedEngine(t)
	_, mnfst, policies := newManifest(t)

	var errCount atomic.Int32
	store.wg.Add(numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := range numWorkers {
		go func() {
			defer wg.Done()
			_, err := guard.UpdateState(ctx, nil, se, mnfst, policies)
			if err != nil {
				errCount.Add(1)
				assert.ErrorIs(err, ErrConcurrentUpdate, "iteration %d", i)
			}
		}()
	}
	wg.Wait()
	assert.Equal(numWorkers-1, int(errCount.Load()))
}

type storeWithSync struct {
	history.Store

	wg sync.WaitGroup
}

func (s *storeWithSync) CompareAndSwap(key string, oldVal []byte, newVal []byte) error {
	s.wg.Done()
	s.wg.Wait()
	return s.Store.CompareAndSwap(key, oldVal, newVal)
}

func TestMetrics(t *testing.T) {
	require := require.New(t)
	a, reg := newTestGuard(t)

	var seed, salt [32]byte
	se, err := seedengine.New(seed[:], salt[:])
	require.NoError(err)

	_, manifestBytes, policies := newManifest(t)

	numGenerations := 12

	s, err := a.GetState(t.Context())
	require.ErrorIs(err, ErrNoState)
	for i := range numGenerations {
		requireGauge(t, reg, i, "iteration %d", i)
		s, err = a.UpdateState(t.Context(), s, se, manifestBytes, policies)
		require.NoError(err, "iteration %d", i)
	}
	requireGauge(t, reg, numGenerations)

	// Simulate a restarted Guard.
	b, reg := newTestGuard(t)
	b.hist = a.hist
	requireGauge(t, reg, 0)

	_, err = b.ResetState(t.Context(), nil, &stubAuthorizer{
		se: se,
		pk: testkeys.ECDSA(t),
	})
	require.NoError(err)
	requireGauge(t, reg, numGenerations)
}

type stubAuthorizer struct {
	se  *seedengine.SeedEngine
	pk  *ecdsa.PrivateKey
	err error
}

func (fa *stubAuthorizer) AuthorizeByManifest(context.Context, *manifest.Manifest) (*seedengine.SeedEngine, *ecdsa.PrivateKey, error) {
	return fa.se, fa.pk, fa.err
}

func TestWatchHistory(t *testing.T) {
	ctx := t.Context()

	for name, tc := range map[string]struct {
		update    []byte
		wantStale bool
	}{
		"valid update": {
			update:    make([]byte, 64),
			wantStale: true,
		},
		"invalid update": {
			update:    []byte{0, 1, 2},
			wantStale: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			store := &storeWithManualUpdates{
				Store:         history.NewAferoStore(&afero.Afero{Fs: afero.NewMemMapFs()}),
				notifications: make(chan []byte),
			}
			hist := history.NewWithStore(slog.Default(), store)
			g := New(hist, prometheus.NewRegistry(), slog.Default())

			_, manifestBytes, policies := newManifest(t)

			se := newSeedEngine(t)
			state, err := g.UpdateState(ctx, nil, se, manifestBytes, policies)
			require.NoError(err)
			require.NotNil(state)

			watchCtx, cancel := context.WithCancel(ctx)
			watcherStopped := make(chan struct{})
			go func() {
				assert.ErrorIs(t, g.WatchHistory(watchCtx), context.Canceled)
				close(watcherStopped)
			}()

			store.notifications <- tc.update

			if tc.wantStale {
				require.EventuallyWithT(func(t *assert.CollectT) {
					assert := assert.New(t)
					staleState, err := g.GetState(ctx)
					assert.ErrorIs(err, ErrStaleState, "watcher update should mark the state stale")
					assert.Same(state, staleState, "GetState should still return the active state")
				}, time.Second, time.Millisecond)
			} else {
				require.Never(func() bool {
					_, err := g.GetState(ctx)
					return errors.Is(err, ErrStaleState)
				}, 100*time.Millisecond, 10*time.Millisecond)
			}

			cancel()
			<-watcherStopped
		})
	}
}

// TestWatchHistoryLateNotifications tests a race condition where the state is updated twice in
// quick succession, but the notifications arrive late. This must not lead to a stale state.
func TestWatchHistoryLateNotifications(t *testing.T) {
	ctx := t.Context()
	require := require.New(t)

	store := &storeWithManualUpdates{
		Store:         history.NewAferoStore(&afero.Afero{Fs: afero.NewMemMapFs()}),
		notifications: make(chan []byte),
	}
	hist := history.NewWithStore(slog.Default(), store)
	g := New(hist, prometheus.NewRegistry(), slog.Default())

	_, manifestBytes, policies := newManifest(t)

	se := newSeedEngine(t)

	watchCtx, cancel := context.WithCancel(ctx)
	watcherStopped := make(chan struct{})
	go func() {
		assert.ErrorIs(t, g.WatchHistory(watchCtx), context.Canceled)
		close(watcherStopped)
	}()

	var notifications [][]byte
	var state *State
	for range 2 {
		nextState, err := g.UpdateState(ctx, state, se, manifestBytes, policies)
		require.NoError(err)
		state = nextState
		latest, err := store.Get("transitions/latest")
		require.NoError(err)
		notifications = append(notifications, latest)
	}

	for _, notification := range notifications {
		store.notifications <- notification
		require.Never(func() bool {
			_, err := g.GetState(ctx)
			return errors.Is(err, ErrStaleState)
		}, 100*time.Millisecond, 10*time.Millisecond)
	}

	cancel()
	<-watcherStopped
}

type storeWithManualUpdates struct {
	history.Store

	notifications chan []byte
}

func (s *storeWithManualUpdates) Watch(string) (<-chan []byte, func(), error) {
	return s.notifications, func() {}, nil
}

// TestBadStoreWatcherIsRestarted tests that a new history watcher is started on failure.
func TestBadStoreWatcherIsRestarted(t *testing.T) {
	fs := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
	store := &badStore{
		Store:      history.NewAferoStore(&afero.Afero{Fs: fs}),
		watchCalls: make(chan string, 10),
	}
	ch := make(chan []byte)
	store.storeUpdates.Store(&ch)
	hist := history.NewWithStore(slog.Default(), store)
	reg := prometheus.NewRegistry()
	a := New(hist, reg, slog.Default())
	clock := &waitingClock{
		FakeClock:  testingclock.NewFakeClock(time.Now()),
		afterCalls: make(chan struct{}, 1),
	}
	a.clock = clock

	ctx, cancel := context.WithCancel(t.Context())
	doneCh := make(chan struct{})
	go func() {
		_ = a.WatchHistory(ctx)
		close(doneCh)
	}()

	t.Cleanup(func() {
		cancel()
		<-doneCh
	})

	// We eventually expect a call to Watch from the goroutine.
	store.WaitForWatchCall(t, time.Second)

	// Simulate a watcher failure. A new watcher should only be created after the clock stepped.
	store.AbortWatch()
	store.EnsureNoWatchCalls(t, 10*time.Millisecond)

	// Wait for the goroutine to request a timer.
	clock.WaitForAfterCall(t, time.Second)
	clock.Step(time.Minute)
	// Advancing the clock should trigger a new watch.
	store.WaitForWatchCall(t, time.Second)
}

type badStore struct {
	history.Store
	storeUpdates atomic.Pointer[chan []byte]
	watchCalls   chan string
}

func (bs *badStore) Watch(s string) (<-chan []byte, func(), error) {
	ch := *bs.storeUpdates.Load()
	bs.watchCalls <- s
	return ch, func() {}, nil
}

func (bs *badStore) WaitForWatchCall(t *testing.T, d time.Duration) {
	select {
	case <-time.After(d):
		require.Fail(t, "no call to Watch")
	case <-bs.watchCalls:
	}
}

func (bs *badStore) EnsureNoWatchCalls(t *testing.T, d time.Duration) {
	select {
	case <-time.After(d):
	case <-bs.watchCalls:
		require.Fail(t, "caught unexpected watch call")
	}
}

func (bs *badStore) AbortWatch() {
	newCh := make(chan []byte)
	ch := bs.storeUpdates.Swap(&newCh)
	close(*ch)
}

type waitingClock struct {
	*testingclock.FakeClock
	afterCalls chan struct{}
}

// After is overridden so that we know when a timer was created. Otherwise, we might be stepping
// the clock before something is waiting on it.
func (c *waitingClock) After(d time.Duration) <-chan time.Time {
	ch := c.FakeClock.After(d)
	c.afterCalls <- struct{}{}
	return ch
}

func (c *waitingClock) WaitForAfterCall(t *testing.T, d time.Duration) {
	select {
	case <-time.After(d):
		require.Fail(t, "no call to After")
	case <-c.afterCalls:
	}
}

func newTestGuard(t *testing.T) (*Guard, *prometheus.Registry) {
	t.Helper()
	store := history.NewAferoStore(&afero.Afero{Fs: afero.NewMemMapFs()})
	hist := history.NewWithStore(slog.Default(), store)
	reg := prometheus.NewRegistry()
	return New(hist, reg, slog.Default()), reg
}

func newManifest(t *testing.T) (*manifest.Manifest, []byte, [][]byte) {
	t.Helper()
	policy := []byte("=== SOME REGO HERE ===")
	policyHash := sha256.Sum256(policy)
	policyHashHex := manifest.NewHexString(policyHash[:])

	workloadOwnerKey := testkeys.ECDSA(t)
	keyDigest := manifest.HashWorkloadOwnerKey(&workloadOwnerKey.PublicKey)

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
	mnfst.WorkloadOwnerKeyDigests = []manifest.HexString{keyDigest}
	mnfstBytes, err := json.Marshal(mnfst)
	require.NoError(t, err)
	return mnfst, mnfstBytes, [][]byte{policy}
}

func newSeedEngine(t *testing.T) *seedengine.SeedEngine {
	t.Helper()
	data := make([]byte, 32)
	se, err := seedengine.New(data, data)
	require.NoError(t, err)
	return se
}

func requireGauge(t *testing.T, reg *prometheus.Registry, val int, fmtArgs ...any) {
	t.Helper()

	expected := fmt.Sprintf(manifestGenerationExpected, val)
	require.NoError(t, testutil.GatherAndCompare(reg, strings.NewReader(expected), "contrast_coordinator_manifest_generation"), fmtArgs...)
}
