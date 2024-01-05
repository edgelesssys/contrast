package snp

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/edgelesssys/nunki/internal/memstore"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	testingclock "k8s.io/utils/clock/testing"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestCachedKDSHTTPClient(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		assert := assert.New(t)

		fakeGetter := &fakeHTTPSGetter{
			content: map[string][]byte{
				"foo": []byte("bar"),
			},
			hits: map[string]int{},
		}
		stepTime := 5 * time.Minute
		testClock := testingclock.NewFakeClock(time.Now())
		ticker := testClock.NewTicker(stepTime)
		client := &cachedKDSHTTPClient{
			HTTPSGetter: fakeGetter,
			gcTicker:    ticker,
			cache:       memstore.New[string, []byte](),
			logger:      slog.Default(),
		}

		res, err := client.Get("foo")
		assert.NoError(err)
		assert.Equal([]byte("bar"), res)
		assert.Equal(1, fakeGetter.hits["foo"])

		// Expect a second call to return the cached value and not increase the hit counter.
		res, err = client.Get("foo")
		assert.NoError(err)
		assert.Equal([]byte("bar"), res)
		assert.Equal(1, fakeGetter.hits["foo"])

		// After the step time, the cache should be invalidated and hit the backend again.
		testClock.Step(stepTime)
		res, err = client.Get("foo")
		assert.NoError(err)
		assert.Equal([]byte("bar"), res)
		assert.Equal(2, fakeGetter.hits["foo"])
	})
	t.Run("Get error", func(t *testing.T) {
		fakeGetter := &fakeHTTPSGetter{
			getErr:  assert.AnError,
			content: map[string][]byte{},
			hits:    map[string]int{},
		}
		testClock := testingclock.NewFakeClock(time.Now())
		ticker := testClock.NewTicker(5 * time.Minute)
		client := &cachedKDSHTTPClient{
			HTTPSGetter: fakeGetter,
			gcTicker:    ticker,
			cache:       memstore.New[string, []byte](),
			logger:      slog.Default(),
		}

		assert := assert.New(t)

		_, err := client.Get("foo")
		assert.Error(err)
		assert.Equal(1, fakeGetter.hits["foo"])
	})
	t.Run("Concurrent access", func(t *testing.T) {
		assert := assert.New(t)

		fakeGetter := &fakeHTTPSGetter{
			content: map[string][]byte{
				"foo": []byte("bar"),
			},
			hits: map[string]int{},
		}
		numGets := 5
		stepTime := 5 * time.Minute
		testClock := testingclock.NewFakeClock(time.Now())
		ticker := testClock.NewTicker(stepTime)
		client := &cachedKDSHTTPClient{
			HTTPSGetter: fakeGetter,
			gcTicker:    ticker,
			cache:       memstore.New[string, []byte](),
			logger:      slog.Default(),
		}

		var wg sync.WaitGroup
		getFunc := func() {
			defer wg.Done()
			res, err := client.Get("foo")
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

		// It's possible that the cache is not yet populated when it is checked by the second Get.
		assert.Less(fakeGetter.hits["foo"], numGets)
	})
}

type fakeHTTPSGetter struct {
	content map[string][]byte
	getErr  error

	hitsMux sync.Mutex
	hits    map[string]int
}

func (f *fakeHTTPSGetter) Get(url string) ([]byte, error) {
	f.hitsMux.Lock()
	defer f.hitsMux.Unlock()
	f.hits[url]++
	return f.content[url], f.getErr
}
