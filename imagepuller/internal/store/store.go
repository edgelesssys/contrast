// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/opencontainers/umoci/oci/layer"
)

// Unpacker allows stubbing out layer.UnpackLayer.
type Unpacker func(root string, layer io.Reader, opt *layer.UnpackOptions) error

// Store OCI image layers and create overlayfs mounts from them.
type Store struct {
	// Unpacker translates an uncompressed tarball to a directory suitable for overlayfs.
	Unpacker Unpacker
	// Root directory where content is stored.
	Root string
}

// PutLayer stores the content of remoteLayer in a directory beneath Root.
// The returned digest can be passed to Store.Mount.
//
// If the function returns no error, a layer will be prepared at Root/shaXXX/0123.../ (not
// necessarily by this invocation, may have been already pulled).
//
// If the function returns an error, there won't be a directory under Root/shaXXX/0123..., and no
// additional storage space will be allocated.
func (s *Store) PutLayer(remoteLayer gcr.Layer) (gcr.Hash, error) {
	digest, err := remoteLayer.Digest()
	if err != nil {
		return gcr.Hash{}, fmt.Errorf("getting layer digest: %w", err)
	}
	algoPath := filepath.Join(s.Root, digest.Algorithm)
	if err := os.MkdirAll(algoPath, 0o755); err != nil {
		return gcr.Hash{}, fmt.Errorf("creating layer dir %q: %w", algoPath, err)
	}
	targetPath := filepath.Join(algoPath, digest.Hex)
	if _, err := os.Stat(targetPath); err == nil {
		// Nothing to do, the layer is already pulled.
		return digest, nil
	}
	stagingDir := filepath.Join(s.Root, stagingDirName)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return gcr.Hash{}, fmt.Errorf("creating staging dir %q: %w", stagingDir, err)
	}
	tempdir, err := os.MkdirTemp(stagingDir, digest.String())
	if err != nil {
		return gcr.Hash{}, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tempdir)

	decompressingReader, err := getDecompressedReader(remoteLayer)
	if err != nil {
		return gcr.Hash{}, fmt.Errorf("picking decompression algorithm: %w", err)
	}
	defer decompressingReader.Close()

	opts := &layer.UnpackOptions{
		OnDiskFormat: layer.OverlayfsRootfs{},
	}
	if err := s.Unpacker(tempdir, decompressingReader, opts); err != nil {
		return gcr.Hash{}, fmt.Errorf("unpacking layer: %w", err)
	}

	if err := os.Rename(tempdir, targetPath); err != nil {
		return gcr.Hash{}, fmt.Errorf("moving unpacked layer: %w", err)
	}
	return digest, nil
}

const stagingDirName = "staging"

// getDecompressedReader returns a reader for the uncompressed layer tarball.
// Its Close method must be called after unpacking finished.
//
// While we could be using remoteLayer.Uncompressed(), the gcr implementation currently uses gzip
// from the stdlib, which causes a significant performance hit compared to pgzip. This is why we
// decompress here.
func getDecompressedReader(remoteLayer gcr.Layer) (io.ReadCloser, error) {
	mediaType, err := remoteLayer.MediaType()
	if err != nil {
		return nil, fmt.Errorf("determining media type: %w", err)
	}
	reader, err := remoteLayer.Compressed()
	if err != nil {
		return nil, fmt.Errorf("getting compressed layer: %w", err)
	}
	switch mediaType {
	case types.DockerLayer, types.OCILayer:
		decompressed, err := pgzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("creating pgzip reader: %w", err)
		}
		return &closer{Reader: decompressed, inner: reader}, nil
	case types.DockerUncompressedLayer, types.OCIUncompressedLayer:
		return &closer{Reader: reader, inner: io.NopCloser(reader)}, nil
	case types.OCILayerZStd:
		decompressed, err := zstd.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("creating zstd reader: %w", err)
		}
		return &closer{Reader: decompressed, inner: reader}, nil
	default:
		return nil, fmt.Errorf("%w: %q", errUnsupportedMediaType, mediaType)
	}
}

var errUnsupportedMediaType = errors.New("unsupported layer media type")

// closer constructs an io.ReadCloser that closes the embedded io.Reader, if present, and an inner io.Closer.
type closer struct {
	io.Reader
	inner io.Closer
}

func (c *closer) Close() error {
	var errs []error
	if outerCloser, ok := c.Reader.(io.Closer); ok {
		errs = append(errs, outerCloser.Close())
	}
	errs = append(errs, c.inner.Close())
	return errors.Join(errs...)
}
