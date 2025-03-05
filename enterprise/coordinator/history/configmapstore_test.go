// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build enterprise

package history

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func coordinatorPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coordinator-0",
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "StatefulSet",
					Name: "coordinator",
					UID:  "00000000-0000-0000-0000-000000000000",
				},
			},
		},
	}
}

func TestGetSet(t *testing.T) {
	require := require.New(t)

	t.Setenv("HOSTNAME", "coordinator-0")
	s, err := NewConfigMapStore(fake.NewSimpleClientset(coordinatorPod()), "test", slog.Default())
	require.NoError(err)

	key := "foo/bar"
	val1 := []byte("val1")
	val2 := []byte("val2")

	_, err = s.Get("invalid-key")
	require.ErrorContains(err, "invalid key")

	x, err := s.Get(key)
	require.True(errors.IsNotFound(err))
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
}

func TestHas(t *testing.T) {
	require := require.New(t)

	t.Setenv("HOSTNAME", "coordinator-0")
	s, err := NewConfigMapStore(fake.NewSimpleClientset(coordinatorPod()), "test", slog.Default())
	require.NoError(err)

	key := "foo/bar"
	val := []byte("val")

	require.False(s.Has(key))

	require.NoError(s.Set(key, val))

	require.True(s.Has(key))
}

func TestCompareAndSwap(t *testing.T) {
	require := require.New(t)

	t.Setenv("HOSTNAME", "coordinator-0")
	s, err := NewConfigMapStore(fake.NewSimpleClientset(coordinatorPod()), "test", slog.Default())
	require.NoError(err)

	key := "foo/bar"
	val1 := []byte("val1")
	val2 := []byte("val2")

	x, err := s.Get(key)
	require.True(errors.IsNotFound(err))
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
}

func TestWatch(t *testing.T) {
	require := require.New(t)

	t.Setenv("HOSTNAME", "coordinator-0")
	s, err := NewConfigMapStore(fake.NewSimpleClientset(coordinatorPod()), "test", slog.Default())
	require.NoError(err)

	key := "foo/bar"
	val1 := []byte("val1")
	val2 := []byte("val2")

	_, _, err = s.Watch("invalid-key")
	require.ErrorContains(err, "invalid key")

	ch, cancel, err := s.Watch(key)
	require.NoError(err)
	defer cancel()

	require.NoError(s.Set(key, val1))
	require.Equal(val1, <-ch)
	require.NoError(s.Set(key, val2))
	require.Equal(val2, <-ch)
}

func TestObjectName(t *testing.T) {
	testCases := map[string]struct {
		key     string
		name    string
		wantErr bool
	}{
		"default": {
			key:     "foo/bar",
			name:    "contrast-store-foo-bar",
			wantErr: false,
		},
		"invalid format": {
			key:     "foo/bar/baz",
			wantErr: true,
		},
		"invalid chars": {
			key:     "***/***",
			wantErr: true,
		},
		"empty key": {
			key:     "",
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			name, err := objectName(tc.key)
			if tc.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
				require.Equal(tc.name, name)
			}
		})
	}
}
