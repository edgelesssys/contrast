// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"errors"
	"io"

	"github.com/containers/storage"
	r "github.com/edgelesssys/contrast/imagepuller/internal/remote"
	"github.com/google/go-containerregistry/pkg/name"
	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/opencontainers/go-digest"
)

type stubStore struct {
	putLayerDigest  digest.Digest
	putLayerErr     error
	putLayerCount   int
	lookupID        string
	lookupSucceeded bool

	storage.Store
}

func (s *stubStore) PutLayer(_, _ string, _ []string, _ string, _ bool, _ *storage.LayerOptions, _ io.Reader) (*storage.Layer, int64, error) {
	s.putLayerCount++
	if s.putLayerErr != nil {
		return nil, 0, s.putLayerErr
	}
	return &storage.Layer{CompressedDigest: s.putLayerDigest}, 0, nil
}

func (s *stubStore) Lookup(_ string) (string, error) {
	if s.lookupID != "" {
		s.lookupSucceeded = true
		return s.lookupID, nil
	}
	return "", errors.New("by default, return a unique error")
}

type stubRemote struct {
	mediaType  types.MediaType
	imageIndex stubImageIndex

	errDescriptor  error
	errRemoteImage error
	errRemoteIndex error

	r.DefaultRemote
}

func (s stubRemote) Head(_ name.Reference, _ ...remote.Option) (*gcr.Descriptor, error) {
	if s.errDescriptor != nil {
		return nil, s.errDescriptor
	}
	if s.mediaType != "" {
		return &gcr.Descriptor{MediaType: s.mediaType}, nil
	}
	return &gcr.Descriptor{MediaType: types.DockerManifestSchema2}, nil
}

func (s stubRemote) Image(_ name.Reference, _ ...remote.Option) (gcr.Image, error) {
	if s.errRemoteImage != nil {
		return nil, s.errRemoteImage
	}
	return nil, nil
}

func (s stubRemote) Index(_ name.Reference, _ ...remote.Option) (gcr.ImageIndex, error) {
	if s.errRemoteIndex != nil {
		return nil, s.errRemoteIndex
	}
	return s.imageIndex, nil
}

type stubImageIndex struct {
	indexManifestIndexManifest *gcr.IndexManifest
	errImage                   error
}

func (s stubImageIndex) MediaType() (types.MediaType, error) {
	return "", nil
}

func (s stubImageIndex) Digest() (gcr.Hash, error) {
	return gcr.Hash{}, nil
}

func (s stubImageIndex) Size() (int64, error) {
	return 0, nil
}

func (s stubImageIndex) IndexManifest() (*gcr.IndexManifest, error) {
	if s.indexManifestIndexManifest != nil {
		return s.indexManifestIndexManifest, nil
	}
	return &gcr.IndexManifest{}, nil
}

func (s stubImageIndex) RawManifest() ([]byte, error) {
	return nil, nil
}

func (s stubImageIndex) Image(_ gcr.Hash) (gcr.Image, error) {
	if s.errImage != nil {
		return nil, s.errImage
	}
	return nil, nil
}

func (s stubImageIndex) ImageIndex(_ gcr.Hash) (gcr.ImageIndex, error) {
	return nil, nil
}

type stubImage struct {
	layersLayers []gcr.Layer
	layersErr    error
	manifestErr  error

	gcr.Image
}

func (s stubImage) Layers() ([]gcr.Layer, error) {
	if s.layersLayers != nil {
		return s.layersLayers, nil
	}
	if s.layersErr != nil {
		return nil, s.layersErr
	}
	return s.Image.Layers()
}

func (s stubImage) Manifest() (*gcr.Manifest, error) {
	if s.manifestErr != nil {
		return nil, s.manifestErr
	}
	return s.Image.Manifest()
}

type stubLayer struct {
	compressedErr error

	gcr.Layer
}

func (s stubLayer) Compressed() (io.ReadCloser, error) {
	return nil, s.compressedErr
}
