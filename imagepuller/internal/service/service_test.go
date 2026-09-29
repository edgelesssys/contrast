// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/edgelesssys/contrast/imagepuller/internal/auth"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"go.podman.io/storage"
)

func TestCleanupOrphanedContainers(t *testing.T) {
	require := require.New(t)

	// live rootfs exists on disk, orphan and gone do not.
	tmp := t.TempDir()
	liveRootfs := filepath.Join(tmp, "live", "rootfs")
	require.NoError(os.MkdirAll(liveRootfs, 0o755))

	store := &stubStore{
		containers: []storage.Container{
			{ID: "live", Metadata: liveRootfs},
			{ID: "orphan", Metadata: filepath.Join(tmp, "orphan", "rootfs")},
			{ID: "no-metadata"},
		},
	}
	s := &ImagePullerService{Logger: slog.New(slog.DiscardHandler)}

	s.cleanupOrphanedContainers(s.Logger, store)

	require.Equal([]string{"orphan"}, store.deleted)
	require.Equal([]string{"orphan"}, store.unmounted)
}

func Test_formatBytes(t *testing.T) {
	tests := []struct {
		bytes uint64
		want  string
	}{
		{
			bytes: 0,
			want:  "0 B",
		},
		{
			bytes: 321,
			want:  "321 B",
		},
		{
			bytes: 4321,
			want:  "4.2 kiB",
		},
		{
			bytes: 54321,
			want:  "53.0 kiB",
		},
		{
			bytes: 654321,
			want:  "639.0 kiB",
		},
		{
			bytes: 7654321,
			want:  "7.3 MiB",
		},
		{
			bytes: 87654321,
			want:  "83.6 MiB",
		},
		{
			bytes: 987654321,
			want:  "941.9 MiB",
		},
		{
			bytes: 9876543210,
			want:  "9.2 GiB",
		},
		{
			bytes: 98765432100,
			want:  "92.0 GiB",
		},
		{
			bytes: 987654321000,
			want:  "919.8 GiB",
		},
		{
			bytes: 9876543210000,
			want:  "9.0 TiB",
		},
		{
			bytes: 98765432100000,
			want:  "89.8 TiB",
		},
		{
			bytes: 10000000000000000000,
			want:  "8.7 EiB",
		},
	}
	for _, tc := range tests {
		name := fmt.Sprintf("%d bytes are %s", tc.bytes, tc.want)
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			got := formatBytes(tc.bytes)
			require.Equal(tc.want, got)
		})
	}
}

func TestPullFromSources(t *testing.T) {
	mirror := auth.Source{Name: "mirror"}
	registry := auth.Source{Name: "registry"}
	errMirror := errors.New("mirror failed")
	errRegistry := errors.New("registry failed")

	testCases := map[string]struct {
		sources     []auth.Source
		results     map[string]error
		cancel      bool
		wantTried   []string
		wantErrs    []error
		wantSuccess bool
	}{
		"mirror succeeds": {
			sources:     []auth.Source{mirror, registry},
			results:     map[string]error{},
			wantTried:   []string{"mirror"},
			wantSuccess: true,
		},
		"falls back to the registry": {
			sources:     []auth.Source{mirror, registry},
			results:     map[string]error{"mirror": errMirror},
			wantTried:   []string{"mirror", "registry"},
			wantSuccess: true,
		},
		"both fail": {
			sources:   []auth.Source{mirror, registry},
			results:   map[string]error{"mirror": errMirror, "registry": errRegistry},
			wantTried: []string{"mirror", "registry"},
			wantErrs:  []error{errMirror, errRegistry},
		},
		"insufficient storage doesn't fall back": {
			sources:   []auth.Source{mirror, registry},
			results:   map[string]error{"mirror": errInsufficientStorage},
			wantTried: []string{"mirror"},
			wantErrs:  []error{errInsufficientStorage},
		},
		"canceled context doesn't fall back": {
			sources:   []auth.Source{mirror, registry},
			results:   map[string]error{"mirror": errMirror},
			cancel:    true,
			wantTried: []string{"mirror"},
			wantErrs:  []error{errMirror},
		},
		"mirror without fallback": {
			sources:   []auth.Source{mirror},
			results:   map[string]error{"mirror": errMirror},
			wantTried: []string{"mirror"},
			wantErrs:  []error{errMirror},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.cancel {
				cancel()
			}

			var tried []string
			layer, err := pullFromSources(ctx, slog.Default(), tc.sources, func(_ *slog.Logger, src auth.Source) (string, error) {
				tried = append(tried, src.Name)
				if err := tc.results[src.Name]; err != nil {
					return "", err
				}
				return "layer-from-" + src.Name, nil
			})

			assert.Equal(tc.wantTried, tried)
			if tc.wantSuccess {
				assert.NoError(err)
				assert.Equal("layer-from-"+tried[len(tried)-1], layer)
				return
			}
			for _, wantErr := range tc.wantErrs {
				assert.ErrorIs(err, wantErr)
			}
		})
	}
}
