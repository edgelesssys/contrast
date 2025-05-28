// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package stateguard

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
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

func requireGauge(t *testing.T, reg *prometheus.Registry, val int, fmtArgs ...any) {
	t.Helper()

	expected := fmt.Sprintf(manifestGenerationExpected, val)
	require.NoError(t, testutil.GatherAndCompare(reg, strings.NewReader(expected), "contrast_coordinator_manifest_generation"), fmtArgs...)
}
