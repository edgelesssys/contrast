// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/containers/storage"
	"github.com/edgelesssys/contrast/imagepuller/internal/test/registry"
	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type StubStore struct {
	putLayerLayer digest.Digest
	putLayerErr   bool

	storage.Store
}

func (s *StubStore) PutLayer(_, _ string, _ []string, _ string, _ bool, _ *storage.LayerOptions, _ io.Reader) (*storage.Layer, int64, error) {
	if s.putLayerErr {
		return nil, 0, errors.New("")
	}
	return &storage.Layer{CompressedDigest: s.putLayerLayer}, 0, nil
}

func TestGetAndVerifyImage_EvilRegistry(t *testing.T) {
	tests := map[string]struct {
		digest  string
		wantErr error
	}{
		"missing digest is rejected": {
			digest:  "",
			wantErr: ErrParseDigest,
		},
		"wrong manifest digest is caught": {
			digest:  registry.WrongManifestDigest(),
			wantErr: ErrRemoteImage,
		},
		"wrong index digest is caught": {
			digest:  registry.WrongIndexDigest(),
			wantErr: ErrRemoteIndex,
		},
		"correct index digest, wrong manifest digest is caught": {
			digest:  registry.IndexForWrongManifestDigest(),
			wantErr: ErrRemoteImage,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			server := httptest.NewServer(registry.New())
			t.Cleanup(server.Close)

			var s ImagePullerService
			_, err := s.getAndVerifyImage(
				t.Context(),
				slog.Default(),
				fmt.Sprintf("%s/busybox:v0.0.1@%s", server.Listener.Addr().String(), tc.digest),
			)

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}

func TestStoreAndVerifyLayers_EvilRegistry(t *testing.T) {
	tests := map[string]struct {
		digest  string
		wantErr error
	}{
		"correct manifest digest, wrong layer digest is caught": {
			digest:  registry.ManifestForWrongBlobDigest(),
			wantErr: ErrValidateLayer,
		},
		"correct index digest, correct manifest digest, wrong layer digest is caught": {
			digest:  registry.IndexForManifestForWrongBlobDigest(),
			wantErr: ErrValidateLayer,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			log := slog.Default()
			server := httptest.NewServer(registry.New())
			t.Cleanup(server.Close)

			s := ImagePullerService{Logger: log, Store: &StubStore{
				putLayerLayer: digest.NewDigestFromBytes(digest.SHA256, []byte{}),
			}}
			remoteImg, err := s.getAndVerifyImage(t.Context(), log, fmt.Sprintf("%s/busybox:v0.0.1@%s", server.Listener.Addr().String(), tc.digest))
			require.NoError(err)

			_, err = s.storeAndVerifyLayers(log, remoteImg)

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}

func TestGetAndVerifyImage(t *testing.T) {
	tests := map[string]struct {
		digest        string
		imageURLNoTag bool
		wantErr       error
		startRegistry bool
	}{
		"digest missing, no tag": {
			digest:        "",
			imageURLNoTag: true,
			wantErr:       ErrParseDigest,
		},
		"digest malformed, no tag": {
			digest:        "sha256:000",
			imageURLNoTag: true,
			wantErr:       ErrParseDigest,
		},
		"digest missing algorithm, no tag": {
			digest:        "0000000000000000000000000000000000000000000000000000000000000000",
			imageURLNoTag: true,
			wantErr:       ErrParseDigest,
		},
		"digest missing, with tag": {
			digest:  "",
			wantErr: ErrParseDigest,
		},
		"digest malformed, with tag": {
			digest:  "sha256:000",
			wantErr: ErrParseDigest,
		},
		"digest missing algorithm, with tag": {
			digest:  "0000000000000000000000000000000000000000000000000000000000000000",
			wantErr: ErrParseDigest,
		},
		"head missing": {
			digest:        "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			wantErr:       ErrDescriptor,
			startRegistry: true,
		},
		"unexpected media type": {
			digest:        registry.UnknownMediaTypeDigest(),
			wantErr:       ErrUnexpectedMediaType,
			startRegistry: true,
		},
		"index valid, linux/amd64 missing": {
			digest:        registry.IndexForMissingPlatformDigest(),
			wantErr:       ErrMissingPlatform,
			startRegistry: true,
		},
		"image success": {
			digest:        registry.ManifestDigest(),
			startRegistry: true,
		},
		"index success": {
			digest:        registry.IndexDigest(),
			startRegistry: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			log := slog.Default()

			var imageURL string
			if tc.imageURLNoTag {
				imageURL = fmt.Sprintf("@%s", tc.digest)
			} else {
				imageURL = fmt.Sprintf("busybox:v0.0.1@%s", tc.digest)
			}

			if tc.startRegistry {
				server := httptest.NewServer(registry.New())
				t.Cleanup(server.Close)

				baseURL := server.Listener.Addr().String()
				imageURL = fmt.Sprintf("%s/%s", baseURL, imageURL)
			}

			var s ImagePullerService
			_, err := s.getAndVerifyImage(context.Background(), log, imageURL)

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}

type StubImage struct {
	Real gcr.Image

	LayersFunc   func() ([]gcr.Layer, error)
	ManifestFunc func() (*gcr.Manifest, error)

	gcr.Image
}

func (s StubImage) Layers() ([]gcr.Layer, error) {
	if s.LayersFunc != nil {
		return s.LayersFunc()
	}
	return s.Real.Layers()
}

func (s StubImage) Manifest() (*gcr.Manifest, error) {
	if s.ManifestFunc != nil {
		return s.ManifestFunc()
	}
	return s.Real.Manifest()
}

type StubLayer struct {
	gcr.Layer
}

func (s StubLayer) Compressed() (io.ReadCloser, error) {
	return nil, errors.New("")
}

func TestStoreAndVerifyLayers(t *testing.T) {
	tests := map[string]struct {
		stubImg     StubImage
		putLayerErr bool
		wantErr     error
	}{
		"layers fails": {
			stubImg: StubImage{LayersFunc: func() ([]gcr.Layer, error) { return nil, errors.New("") }},
			wantErr: ErrObtainingLayers,
		},
		"manifest fails": {
			stubImg: StubImage{ManifestFunc: func() (*gcr.Manifest, error) { return nil, errors.New("") }},
			wantErr: ErrObtainingManifest,
		},
		"compressed fails": {
			stubImg: StubImage{LayersFunc: func() ([]gcr.Layer, error) {
				return []gcr.Layer{StubLayer{}}, nil
			}},
			wantErr: ErrReadLayer,
		},
		"put layer fails": {
			stubImg:     StubImage{},
			putLayerErr: true,
			wantErr:     ErrPutLayer,
		},
		"success": {
			stubImg: StubImage{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			log := slog.Default()
			server := httptest.NewServer(registry.New())
			t.Cleanup(server.Close)

			t.Log(registry.BlobDigest())

			store := &StubStore{
				putLayerLayer: digest.NewDigestFromEncoded(digest.SHA256, registry.BlobDigest()[7:]),
				putLayerErr:   tc.putLayerErr,
			}
			s := ImagePullerService{Logger: log, Store: store}
			realImg, err := s.getAndVerifyImage(t.Context(), log, fmt.Sprintf("%s/busybox:v0.0.1@%s", server.Listener.Addr().String(), registry.ManifestDigest()))
			require.NoError(err)

			tc.stubImg.Real = realImg
			_, err = s.storeAndVerifyLayers(log, tc.stubImg)

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}
