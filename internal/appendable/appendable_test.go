// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package appendable_test

import (
	"sync"
	"testing"

	"github.com/edgelesssys/contrast/internal/appendable"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestAppendable(t *testing.T) {
	t.Run("new appendable is empty", func(t *testing.T) {
		assert := assert.New(t)

		a := appendable.Appendable[string]{}
		assert.Empty(a.All())
	})

	t.Run("append and all", func(t *testing.T) {
		assert := assert.New(t)

		a := appendable.Appendable[string]{}
		a.Append("foo")
		a.Append("bar")

		values := a.All()
		assert.Len(values, 2)
		assert.Contains(values, "foo")
		assert.Contains(values, "bar")
	})

	t.Run("latest", func(t *testing.T) {
		assert := assert.New(t)

		a := appendable.Appendable[string]{}
		a.Append("foo")
		a.Append("bar")

		v, err := a.Latest()
		assert.NoError(err)
		assert.Equal("bar", v)
	})

	t.Run("latest empty", func(t *testing.T) {
		assert := assert.New(t)

		a := appendable.Appendable[string]{}
		_, err := a.Latest()
		assert.Error(err)
	})
}

func TestAppendableConcurrent(t *testing.T) {
	t.Run("append all latest", func(t *testing.T) {
		assert := assert.New(t)

		a := appendable.Appendable[string]{}

		var wg sync.WaitGroup

		appendElem := func(e string) {
			defer wg.Done()
			a.Append(e)
		}
		all := func() {
			defer wg.Done()
			_ = a.All()
		}
		latest := func() {
			defer wg.Done()
			_, _ = a.Latest()
		}

		wg.Add(18)
		go latest()
		go latest()
		go latest()
		go appendElem("foo")
		go appendElem("bar")
		go appendElem("baz")
		go all()
		go all()
		go all()
		go appendElem("foo")
		go appendElem("bar")
		go appendElem("baz")
		go latest()
		go latest()
		go latest()
		go all()
		go all()
		go all()
		wg.Wait()

		values := a.All()
		assert.Len(values, 6)
		assert.Contains(values, "foo")
		assert.Contains(values, "bar")
		assert.Contains(values, "baz")
	})
}
