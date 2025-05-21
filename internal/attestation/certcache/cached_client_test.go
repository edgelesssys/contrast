// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package certcache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/internal/memstore"
	sevtrust "github.com/google/go-sev-guest/verify/trust"
	tdxtrust "github.com/google/go-tdx-guest/verify/trust"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"k8s.io/utils/clock"
	testingclock "k8s.io/utils/clock/testing"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

const crlURLMatch string = "https://kdsintf.amd.com/vcek/v1/test/crl"

func TestMemcachedHTTPSGetter(t *testing.T) {
	stepTime := 5 * time.Minute
	testClock := testingclock.NewFakeClock(time.Now())
	ticker := testClock.NewTicker(stepTime)

	t.Run("Get VCEK by request and from cache", func(t *testing.T) {
		assert := assert.New(t)
		fakeGetter, client := getFakeHTTPSGetters(ticker)

		_, res, err := client.Get("foo")
		assert.NoError(err)
		assert.Equal([]byte("bar"), res)
		assert.Equal(1, fakeGetter.hits["foo"])

		// Expect a second call to return the cached value and not increase the hit counter.
		_, res, err = client.Get("foo")
		assert.NoError(err)
		assert.Equal([]byte("bar"), res)
		assert.Equal(1, fakeGetter.hits["foo"])

		// After the step time, the cache should be invalidated and hit the backend again.
		testClock.Step(stepTime)
		_, res, err = client.Get("foo")
		assert.NoError(err)
		assert.Equal([]byte("bar"), res)
		assert.Equal(2, fakeGetter.hits["foo"])
	})

	t.Run("VCEK request fails and VCEK not in cache", func(t *testing.T) {
		assert := assert.New(t)
		fakeGetter, client := getFakeHTTPSGetters(ticker)

		// Simulate a request failure by returning an error
		fakeGetter.getErr = errors.New("VCEK request failure")

		_, _, err := client.Get("foo")
		assert.Error(err)
		assert.Equal(1, fakeGetter.hits["foo"])
	})

	t.Run("Check CRLs are still requested after caching", func(t *testing.T) {
		assert := assert.New(t)
		fakeGetter, client := getFakeHTTPSGetters(ticker)

		_, res, err := client.Get(crlURLMatch)
		assert.NoError(err)
		assert.Equal([]byte("bar"), res)
		assert.Equal(1, fakeGetter.hits[crlURLMatch])

		// Even after the CRL is cached, the CRL should be requested(hit counter increase).
		_, res, err = client.Get(crlURLMatch)
		assert.NoError(err)
		assert.Equal([]byte("bar"), res)
		assert.Equal(2, fakeGetter.hits[crlURLMatch])
	})

	t.Run("Check CRLs can be loaded by cache when request fails", func(t *testing.T) {
		assert := assert.New(t)
		fakeGetter, client := getFakeHTTPSGetters(ticker)

		// Preload CRL into the cache
		data, err := json.Marshal(cacheEntry{Body: []byte("bar")})
		assert.NoError(err)
		client.cache.Set(crlURLMatch, data)
		fakeGetter.getErr = errors.New("CRL request failure")

		// The CRL should be loaded from the cache and client.Get() won't result in an error
		_, res, err := client.Get(crlURLMatch)
		assert.NoError(err)
		assert.Equal([]byte("bar"), res)
		assert.Equal(1, fakeGetter.hits[crlURLMatch])
	})

	t.Run("CRL request fails and CRL not in cache", func(t *testing.T) {
		assert := assert.New(t)
		fakeGetter, client := getFakeHTTPSGetters(ticker)

		fakeGetter.getErr = errors.New("CRL request failure")

		// No CRL cache and request failure results in error
		_, _, err := client.Get(crlURLMatch)
		assert.Error(err)
		assert.Equal(1, fakeGetter.hits[crlURLMatch])
	})

	t.Run("Concurrent access", func(t *testing.T) {
		assert := assert.New(t)
		_, client := getFakeHTTPSGetters(ticker)
		numGets := 5

		var wg sync.WaitGroup
		getFunc := func() {
			defer wg.Done()
			_, res, err := client.Get("foo")
			assert.NoError(err)
			assert.Equal([]byte("bar"), res)
		}

		wg.Add(numGets)
		go getFunc()
		go getFunc()
		go getFunc()
		go getFunc()
		go getFunc()
		wg.Wait()
	})

	t.Run("Request header is cached", func(t *testing.T) {
		assert := assert.New(t)
		fakeGetter, client := getFakeHTTPSGetters(ticker)

		header, res, err := client.Get("foo")
		assert.NoError(err)
		assert.Equal([]byte("bar"), res)
		assert.Equal(1, fakeGetter.hits["foo"])
		assert.Equal(map[string][]string{"bar": {"baz"}}, header)

		// Expect a second call to return both the cached header and body and not increase the hit counter.
		header, res, err = client.Get("foo")
		assert.NoError(err)
		assert.Equal([]byte("bar"), res)
		assert.Equal(1, fakeGetter.hits["foo"])
		assert.Equal(map[string][]string{"bar": {"baz"}}, header)
	})
}

func TestContextCancellation(t *testing.T) {
	stepTime := 5 * time.Minute
	testClock := testingclock.NewFakeClock(time.Now())
	ticker := testClock.NewTicker(stepTime)

	fakeGetter := &fakeHTTPSGetter{
		content: map[string][]byte{},
		hits:    map[string]int{},
	}

	getter := &CachedHTTPSGetter{
		ContextHTTPSGetter: fakeGetter,
		gcTicker:           ticker,
		cache:              memstore.New[string, []byte](),
		logger:             slog.Default(),
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := getter.GetContext(ctx, crlURLMatch)
	require.ErrorIs(t, err, context.Canceled)
}

// Ensure CachedHTTPSGetter implements the expected interfaces.
var (
	_ = tdxtrust.HTTPSGetter(&CachedHTTPSGetter{})
	_ = tdxtrust.ContextHTTPSGetter(&CachedHTTPSGetter{})
	_ = sevtrust.HTTPSGetter(&CachedHTTPSGetterSNP{})
	_ = sevtrust.ContextHTTPSGetter(&CachedHTTPSGetterSNP{})
)

type fakeHTTPSGetter struct {
	header  map[string]map[string][]string
	content map[string][]byte
	getErr  error

	hitsMux sync.Mutex
	hits    map[string]int
}

// Returns the fakeHTTPSGetter for test assertions and its wrapper CachedHTTPSGetter.
func getFakeHTTPSGetters(ticker clock.Ticker) (*fakeHTTPSGetter, *CachedHTTPSGetter) {
	fakeGetter := &fakeHTTPSGetter{
		header: map[string]map[string][]string{
			"foo": {"bar": {"baz"}},
		},
		content: map[string][]byte{
			"foo":       []byte("bar"),
			crlURLMatch: []byte("bar"),
		},
		hits: map[string]int{},
	}

	return fakeGetter, &CachedHTTPSGetter{
		ContextHTTPSGetter: fakeGetter,
		gcTicker:           ticker,
		cache:              memstore.New[string, []byte](),
		logger:             slog.Default(),
	}
}

func (f *fakeHTTPSGetter) GetContext(ctx context.Context, url string) (map[string][]string, []byte, error) {
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}
	f.hitsMux.Lock()
	defer f.hitsMux.Unlock()
	f.hits[url]++
	return f.header[url], f.content[url], f.getErr
}
