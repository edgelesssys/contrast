// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package fsstore

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestStore(t *testing.T) {
	newTestStore := func() (*Store, func() error) {
		handler := &testHandler{
			inner: slog.Default().Handler(),
		}
		fs := &afero.Afero{
			Fs: afero.NewMemMapFs(),
		}
		return &Store{fs: fs, logger: slog.New(handler)}, handler.loggedErr
	}

	t.Run("new store is empty", func(t *testing.T) {
		assert := assert.New(t)

		s, loggedErr := newTestStore()
		assert.Empty(s.GetAll())
		assert.NoError(loggedErr())
	})

	t.Run("set and get", func(t *testing.T) {
		assert := assert.New(t)

		s, loggedErr := newTestStore()
		s.Set("foo", []byte("bar"))
		s.Set("bar", []byte("baz"))

		v, ok := s.Get("foo")
		assert.True(ok)
		assert.Equal([]byte("bar"), v)

		v, ok = s.Get("bar")
		assert.True(ok)
		assert.Equal([]byte("baz"), v)

		_, ok = s.Get("baz")
		assert.False(ok)

		assert.NoError(loggedErr())
	})

	t.Run("get all", func(t *testing.T) {
		assert := assert.New(t)

		s, loggedErr := newTestStore()
		s.Set("foo", []byte("bar"))
		s.Set("bar", []byte("baz"))

		values := s.GetAll()
		assert.Len(values, 2)
		assert.Contains(values, []byte("bar"))
		assert.Contains(values, []byte("baz"))
		assert.NoError(loggedErr())
	})

	t.Run("clear elements", func(t *testing.T) {
		assert := assert.New(t)

		s, loggedErr := newTestStore()
		s.Set("foo", []byte("bar"))
		s.Set("bar", []byte("baz"))

		s.Clear()
		assert.Empty(s.GetAll())
		assert.NoError(loggedErr())
	})

	t.Run("clear empty", func(t *testing.T) {
		assert := assert.New(t)

		s, loggedErr := newTestStore()
		s.Clear()
		assert.Empty(s.GetAll())
		assert.NoError(loggedErr())
	})
}

type testHandler struct {
	err   error
	inner slog.Handler
}

func (h *testHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Level == slog.LevelError {
		h.err = errors.Join(h.err, errors.New(record.Message))
	}
	return h.inner.Handle(ctx, record)
}

func (h *testHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *testHandler) WithAttrs(_ []slog.Attr) slog.Handler         { return h }
func (h *testHandler) WithGroup(_ string) slog.Handler              { return h }
func (h *testHandler) loggedErr() error                             { return h.err }
