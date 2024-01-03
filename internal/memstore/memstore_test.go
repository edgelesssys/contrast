package memstore_test

import (
	"sync"
	"testing"

	"github.com/edgelesssys/nunki/internal/memstore"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestStore(t *testing.T) {
	t.Run("new store is empty", func(t *testing.T) {
		assert := assert.New(t)

		s := memstore.New[string, int]()
		assert.Equal(0, len(s.GetAll()))
	})

	t.Run("set and get", func(t *testing.T) {
		assert := assert.New(t)

		s := memstore.New[string, int]()
		s.Set("foo", 1)
		s.Set("bar", 2)

		v, ok := s.Get("foo")
		assert.True(ok)
		assert.Equal(1, v)

		v, ok = s.Get("bar")
		assert.True(ok)
		assert.Equal(2, v)

		v, ok = s.Get("baz")
		assert.False(ok)
		assert.Equal(0, v)
	})

	t.Run("get all", func(t *testing.T) {
		assert := assert.New(t)

		s := memstore.New[string, int]()
		s.Set("foo", 1)
		s.Set("bar", 2)

		values := s.GetAll()
		assert.Equal(2, len(values))
		assert.Contains(values, 1)
		assert.Contains(values, 2)
	})
}

func TestStoreConcurrent(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		assert := assert.New(t)

		s := memstore.New[string, int]()

		var wg sync.WaitGroup

		set := func(key string, value int) {
			defer wg.Done()
			s.Set(key, value)
		}
		get := func(key string) {
			defer wg.Done()
			// we don't know if the value exists or not,
			// we just get it to provoke a race
			_, _ = s.Get(key)
		}

		wg.Add(15)
		go get("foo")
		go get("bar")
		go get("baz")
		go set("foo", 1)
		go set("bar", 2)
		go set("baz", 3)
		go get("foo")
		go get("bar")
		go get("baz")
		go set("foo", 4)
		go set("bar", 5)
		go set("baz", 6)
		go get("foo")
		go get("bar")
		go get("baz")
		wg.Wait()

		// we don't know what the final values are,
		// but keys should exist.
		_, ok := s.Get("foo")
		assert.True(ok)
		_, ok = s.Get("bar")
		assert.True(ok)
		_, ok = s.Get("baz")
		assert.True(ok)
	})

	t.Run("get all", func(t *testing.T) {
		assert := assert.New(t)

		s := memstore.New[string, int]()

		var wg sync.WaitGroup

		set := func(key string, value int) {
			defer wg.Done()
			s.Set(key, value)
		}
		get := func(key string) {
			defer wg.Done()
			_, _ = s.Get(key)
		}
		getAll := func() {
			defer wg.Done()
			_ = s.GetAll()
		}

		wg.Add(16)
		go get("foo")
		go get("bar")
		go set("foo", 1)
		go set("bar", 2)
		go getAll()
		go getAll()
		go getAll()
		go getAll()
		go get("foo")
		go get("bar")
		go set("baz", 3)
		go set("pil", 4)
		go getAll()
		go getAll()
		go getAll()
		go getAll()
		wg.Wait()

		assert.Equal(4, len(s.GetAll()))
	})
}
