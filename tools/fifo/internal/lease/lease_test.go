// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package lease

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	testingclock "k8s.io/utils/clock/testing"
)

func TestLeaseFlow(t *testing.T) {
	require := require.New(t)
	client := &fakeClient{}
	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
	clock := testingclock.NewFakeClock(time.Now())

	lease := New("test", "me", 60*time.Second, client, logger)
	lease.clock = clock

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	// Acquiring with empty state should work.
	require.NoError(lease.Acquire(ctx))

	// Releasing without holding should fail.
	other := New("test", "other-holder", 60*time.Second, client, logger)
	other.clock = clock
	require.Error(other.Release(ctx))

	// Releasing should work.
	require.NoError(lease.Release(ctx))

	// Releasing twice should fail.
	require.Error(lease.Release(ctx))
}

func TestConcurrentAcquire(t *testing.T) {
	assert := assert.New(t)
	wg := sync.WaitGroup{}
	client := &fakeClient{}
	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
	clock := testingclock.NewFakeClock(time.Now())

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	var gauge atomic.Int32

	for i := range 21 {
		wg.Go(func() {
			holder := fmt.Sprintf("holder-%02d", i)
			lease := New("test", holder, 60*time.Second, client, logger.With("candidate", holder))
			lease.clock = clock
			lease.pollInterval = time.Millisecond

			// The assertions below look a bit odd because we need to make sure
			// - that this goroutine exits instead of blocking,
			// - that waiting goroutines are notified about the failure and
			// - that the test case itself fails.
			// require.NoError and friends don't seem to affect the other goroutines spawned, so we
			// just register a test failure, cancel the context to stop other goroutines currently
			// in Acquire, and plain return to quit this goroutine.
			if !assert.NoError(lease.Acquire(ctx), holder) {
				cancel()
				return
			}
			if !assert.True(gauge.CompareAndSwap(0, 1), holder) {
				cancel()
				return
			}
			time.Sleep(3 * time.Millisecond)
			if !assert.True(gauge.CompareAndSwap(1, 0), holder) {
				cancel()
				return
			}
			if !assert.NoError(lease.Release(ctx), holder) {
				cancel()
				return
			}
		})
	}

	wg.Wait()
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
