// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package history

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestAferoStore(t *testing.T) {
	t.Run("memmap", func(t *testing.T) {
		suite(t, func(_ *testing.T) Store {
			return NewAferoStore(&afero.Afero{Fs: afero.NewMemMapFs()})
		})
	})

	t.Run("tmpdir", func(t *testing.T) {
		suite(t, func(t *testing.T) Store {
			return NewAferoStore(&afero.Afero{Fs: afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())})
		})
	})
}

func suite(t *testing.T, storeFactory func(t *testing.T) Store) {
	t.Run("Get and Set", func(t *testing.T) {
		require := require.New(t)
		s := storeFactory(t)

		key := "foo/bar"
		val1 := []byte("val1")
		val2 := []byte("val2")

		_, err := s.Get("invalid-key")
		require.ErrorContains(err, "invalid key")

		x, err := s.Get(key)
		require.ErrorIs(err, os.ErrNotExist)
		require.Empty(x)

		require.NoError(s.Set(key, val1))

		y, err := s.Get(key)
		require.NoError(err)
		require.Equal(val1, y)

		require.ErrorContains(s.Set("invalid-key", nil), "invalid key")

		require.NoError(s.Set(key, val2))

		z, err := s.Get(key)
		require.NoError(err)
		require.Equal(val2, z)
	})

	t.Run("Has", func(t *testing.T) {
		require := require.New(t)
		s := storeFactory(t)

		key := "foo/bar"
		val := []byte("val")

		require.False(s.Has(key))

		require.NoError(s.Set(key, val))

		require.True(s.Has(key))
	})

	t.Run("CompareAndSwap", func(t *testing.T) {
		require := require.New(t)
		s := storeFactory(t)

		key := "foo/bar"
		val1 := []byte("val1")
		val2 := []byte("val2")

		x, err := s.Get(key)
		require.ErrorIs(err, os.ErrNotExist)
		require.Empty(x)

		require.ErrorContains(s.CompareAndSwap("invalid-key", nil, nil), "invalid key")
		require.Error(s.CompareAndSwap(key, val1, val2))

		require.NoError(s.CompareAndSwap(key, nil, val1))

		y, err := s.Get(key)
		require.NoError(err)
		require.Equal(val1, y)

		require.Error(s.CompareAndSwap(key, nil, val1))

		require.NoError(s.CompareAndSwap(key, val1, val2))

		z, err := s.Get(key)
		require.NoError(err)
		require.Equal(val2, z)
	})
}
