// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"crypto/hpke"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPendingKeys(t *testing.T) {
	newKey := func(t *testing.T) hpke.PrivateKey {
		t.Helper()
		key, err := hpke.MLKEM768X25519().GenerateKey()
		require.NoError(t, err)
		return key
	}
	id := func(b byte) [32]byte { return [32]byte{b} }

	t.Run("a key is handed out once", func(t *testing.T) {
		require := require.New(t)
		keys := NewPendingKeys()
		key := newKey(t)

		keys.Put(id(1), key)
		got, ok := keys.Take(id(1))
		require.True(ok)
		require.Equal(key, got)

		_, ok = keys.Take(id(1))
		require.False(ok)
	})

	t.Run("unknown nonce", func(t *testing.T) {
		_, ok := NewPendingKeys().Take(id(1))
		require.False(t, ok)
	})

	t.Run("keys expire", func(t *testing.T) {
		require := require.New(t)
		now := time.Now()
		keys := NewPendingKeys()
		keys.now = func() time.Time { return now }

		keys.Put(id(1), newKey(t))
		now = now.Add(pendingKeyTTL / 2)
		keys.Put(id(2), newKey(t))

		now = now.Add(pendingKeyTTL / 2)
		_, ok := keys.Take(id(1))
		require.False(ok, "key must be gone once its TTL elapsed")
		_, ok = keys.Take(id(2))
		require.True(ok, "key must be kept until its TTL elapsed")
		require.Empty(keys.keys)
	})

	t.Run("size is bounded", func(t *testing.T) {
		require := require.New(t)
		keys := NewPendingKeys()
		keys.max = 2
		key := newKey(t)

		keys.Put(id(1), key)
		keys.Put(id(2), key)
		keys.Put(id(3), key)
		require.Len(keys.keys, 2)

		_, ok := keys.Take(id(1))
		require.False(ok, "the oldest key must be dropped first")
		_, ok = keys.Take(id(2))
		require.True(ok)
		_, ok = keys.Take(id(3))
		require.True(ok)
	})

	t.Run("a nonce holds one key", func(t *testing.T) {
		require := require.New(t)
		keys := NewPendingKeys()
		keys.max = 2
		first, second := newKey(t), newKey(t)

		keys.Put(id(1), first)
		keys.Put(id(1), second)
		require.Len(keys.keys, 1)

		got, ok := keys.Take(id(1))
		require.True(ok)
		require.Equal(second, got)
	})
}
