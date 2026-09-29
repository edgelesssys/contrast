// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edgelesssys/contrast/imagepuller/internal/remote"
	"github.com/edgelesssys/contrast/imagepuller/internal/test/registry"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/opencontainers/go-digest"

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

// TestPullLayersMirrorFallback pulls through a mirror that has the manifest but lost the layers,
// like the k3s embedded registry after an image was removed from the node.
func TestPullLayersMirrorFallback(t *testing.T) {
	testCases := map[string]struct {
		fallback bool
		wantErr  bool
	}{
		"without fallback the pull fails":              {wantErr: true},
		"with fallback the registry serves the layers": {fallback: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			log := slog.Default()

			var registryAuth []string
			upstream := registry.New()
			registrySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				registryAuth = append(registryAuth, r.Header.Get("Authorization"))
				upstream.ServeHTTP(w, r)
			}))
			t.Cleanup(registrySrv.Close)

			var mirrorAuth []string
			mirrored := registry.New()
			mirrorSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mirrorAuth = append(mirrorAuth, r.Header.Get("Authorization"))
				if strings.Contains(r.URL.Path, "/blobs/") {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				mirrored.ServeHTTP(w, r)
			}))
			t.Cleanup(mirrorSrv.Close)

			cfg := auth.Config{Registries: map[string]auth.Registry{
				".": {
					AuthConfig:     authn.AuthConfig{Username: "mirror-user", Password: "mirror-password"},
					Mirror:         mirrorSrv.URL,
					MirrorFallback: tc.fallback,
				},
			}}
			imageURL := fmt.Sprintf("%s/busybox:v0.0.1@%s", registrySrv.Listener.Addr().String(), registry.ManifestDigest())
			sources, err := cfg.SourcesFor(imageURL, log)
			require.NoError(err)

			store := &stubStore{
				putLayerDigest: digest.NewDigestFromEncoded(digest.SHA256, registry.BlobDigest()[7:]),
				graphRoot:      t.TempDir(),
			}
			s := ImagePullerService{Logger: log, Store: store, Remote: remote.DefaultRemote{}}
			_, err = pullFromSources(t.Context(), log, sources, func(log *slog.Logger, src auth.Source) (string, error) {
				return s.pullLayers(t.Context(), log, imageURL, src)
			})

			assert.NotEmpty(mirrorAuth, "the pull must go through the mirror first")
			if tc.wantErr {
				assert.Error(err)
				assert.Empty(registryAuth, "without fallback the registry must not be contacted")
				return
			}
			require.NoError(err)
			assert.NotEmpty(registryAuth, "the fallback must contact the registry")
			assert.Contains(mirrorAuth, "Basic "+base64.StdEncoding.EncodeToString([]byte("mirror-user:mirror-password")), "the mirror must receive its credentials")
			for _, header := range registryAuth {
				assert.Empty(header, "the mirror's credentials must not reach the registry")
			}
		})
	}
}
