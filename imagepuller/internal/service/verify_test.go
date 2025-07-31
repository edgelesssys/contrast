// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/containers/storage"
	"github.com/edgelesssys/contrast/imagepuller/internal/test/registry"
	"github.com/google/go-containerregistry/pkg/name"
	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
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
		return nil, 0, assert.AnError
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

			s := ImagePullerService{
				Remote: DefaultRemote{},
			}
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

			s := ImagePullerService{
				Logger: log,
				Store: &StubStore{
					putLayerLayer: digest.NewDigestFromBytes(digest.SHA256, []byte{}),
				},
				Remote: DefaultRemote{},
			}
			remoteImg, err := s.getAndVerifyImage(t.Context(), log, fmt.Sprintf("%s/busybox:v0.0.1@%s", server.Listener.Addr().String(), tc.digest))
			require.NoError(err)

			_, err = s.storeAndVerifyLayers(log, remoteImg)

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}

var zeroDigest = "0000000000000000000000000000000000000000000000000000000000000000"

type StubRemote struct {
	mediaType  types.MediaType
	imageIndex StubImageIndex

	headErr  bool
	imageErr bool
	indexErr bool

	DefaultRemote
}

type StubImageIndex struct {
	includeLinux bool
	imageErr     bool
}

func (s StubImageIndex) MediaType() (types.MediaType, error) {
	return "", nil
}

func (s StubImageIndex) Digest() (gcr.Hash, error) {
	return gcr.Hash{}, nil
}

func (s StubImageIndex) Size() (int64, error) {
	return 0, nil
}

func (s StubImageIndex) IndexManifest() (*gcr.IndexManifest, error) {
	if s.includeLinux {
		return &gcr.IndexManifest{
			Manifests: []gcr.Descriptor{{
				Platform: &gcr.Platform{
					OS:           "linux",
					Architecture: "amd64",
				},
				Digest: gcr.Hash{
					Algorithm: "sha256",
					Hex:       "",
				},
			}},
		}, nil
	}
	return &gcr.IndexManifest{}, nil
}

func (s StubImageIndex) RawManifest() ([]byte, error) {
	return nil, nil
}

func (s StubImageIndex) Image(_ gcr.Hash) (gcr.Image, error) {
	if s.imageErr {
		return nil, assert.AnError
	}
	return nil, nil
}

func (s StubImageIndex) ImageIndex(_ gcr.Hash) (gcr.ImageIndex, error) {
	return nil, nil
}

type StubIndexManifest struct {
	gcr.IndexManifest
}

func (s StubRemote) Head(_ name.Reference, _ ...remote.Option) (*gcr.Descriptor, error) {
	if s.headErr {
		return nil, assert.AnError
	}
	if s.mediaType != "" {
		return &gcr.Descriptor{MediaType: s.mediaType}, nil
	}
	return &gcr.Descriptor{MediaType: types.DockerManifestSchema2}, nil
}

func (s StubRemote) Image(_ name.Reference, _ ...remote.Option) (gcr.Image, error) {
	if s.imageErr {
		return nil, assert.AnError
	}
	return nil, nil
}

func (s StubRemote) Index(_ name.Reference, _ ...remote.Option) (gcr.ImageIndex, error) {
	if s.indexErr {
		return nil, assert.AnError
	}
	return s.imageIndex, nil
}

func TestGetAndVerifyImage(t *testing.T) {
	tests := map[string]struct {
		digest        string
		emptyDigest   bool
		imageRefNoTag bool
		wantErr       error
		stubRemote    StubRemote
	}{
		"digest missing, no tag": {
			emptyDigest:   true,
			imageRefNoTag: true,
			wantErr:       ErrParseDigest,
		},
		"digest malformed, no tag": {
			digest:        "sha256:000",
			imageRefNoTag: true,
			wantErr:       ErrParseDigest,
		},
		"digest missing algorithm, no tag": {
			digest:        zeroDigest,
			imageRefNoTag: true,
			wantErr:       ErrParseDigest,
		},
		"digest missing, with tag": {
			emptyDigest: true,
			wantErr:     ErrParseDigest,
		},
		"digest malformed, with tag": {
			digest:  "sha256:000",
			wantErr: ErrParseDigest,
		},
		"digest missing algorithm, with tag": {
			digest:  zeroDigest,
			wantErr: ErrParseDigest,
		},
		"head request error": {
			wantErr:    ErrDescriptor,
			stubRemote: StubRemote{headErr: true},
		},
		"remote image failure": {
			wantErr:    ErrRemoteImage,
			stubRemote: StubRemote{imageErr: true, mediaType: types.DockerManifestSchema2},
		},
		"remote index failure": {
			wantErr:    ErrRemoteIndex,
			stubRemote: StubRemote{indexErr: true, mediaType: types.DockerManifestList},
		},
		"unexpected media type": {
			wantErr:    ErrUnexpectedMediaType,
			stubRemote: StubRemote{mediaType: types.DockerForeignLayer},
		},
		"index valid, linux/amd64 missing": {
			wantErr:    ErrMissingPlatform,
			stubRemote: StubRemote{mediaType: types.DockerManifestList},
		},
		"image success": {
			stubRemote: StubRemote{mediaType: types.DockerManifestSchema2},
		},
		"index remote image failure": {
			wantErr: ErrRemoteImage,
			stubRemote: StubRemote{
				mediaType:  types.DockerManifestList,
				imageIndex: StubImageIndex{includeLinux: true, imageErr: true},
			},
		},
		"index success": {
			stubRemote: StubRemote{
				mediaType:  types.DockerManifestList,
				imageIndex: StubImageIndex{includeLinux: true},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			log := slog.Default()

			var digest string
			if tc.digest == "" && !tc.emptyDigest {
				digest = fmt.Sprintf("sha256:%s", zeroDigest)
			} else {
				digest = tc.digest
			}

			var imageRef string
			if tc.imageRefNoTag {
				imageRef = fmt.Sprintf("busybox@%s", digest)
			} else {
				imageRef = fmt.Sprintf("busybox:v0.0.1@%s", digest)
			}

			s := ImagePullerService{Remote: tc.stubRemote}
			_, err := s.getAndVerifyImage(context.Background(), log, imageRef)

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}

type StubImage struct {
	layersLayers []gcr.Layer
	layersErr    bool
	manifestErr  bool

	gcr.Image
}

func (s StubImage) Layers() ([]gcr.Layer, error) {
	if s.layersLayers != nil {
		return s.layersLayers, nil
	}
	if s.layersErr {
		return nil, assert.AnError
	}
	return s.Image.Layers()
}

func (s StubImage) Manifest() (*gcr.Manifest, error) {
	if s.manifestErr {
		return nil, assert.AnError
	}
	return s.Image.Manifest()
}

type StubLayer struct {
	gcr.Layer
}

func (s StubLayer) Compressed() (io.ReadCloser, error) {
	return nil, assert.AnError
}

func TestStoreAndVerifyLayers(t *testing.T) {
	tests := map[string]struct {
		stubImg     StubImage
		stubRemote  StubRemote
		putLayerErr bool
		wantErr     error
	}{
		"layers fails": {
			stubImg: StubImage{layersErr: true},
			wantErr: ErrObtainingLayers,
		},
		"manifest fails": {
			stubImg: StubImage{manifestErr: true},
			wantErr: ErrObtainingManifest,
		},
		"compressed fails": {
			stubImg: StubImage{layersLayers: []gcr.Layer{StubLayer{}}},
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
			s := ImagePullerService{Logger: log, Store: store, Remote: tc.stubRemote.DefaultRemote}
			realImg, err := s.getAndVerifyImage(t.Context(), log, fmt.Sprintf("%s/busybox:v0.0.1@%s", server.Listener.Addr().String(), registry.ManifestDigest()))
			require.NoError(err)

			tc.stubImg.Image = realImg
			_, err = s.storeAndVerifyLayers(log, tc.stubImg)

			assert.ErrorIs(err, tc.wantErr)
		})
	}
}
