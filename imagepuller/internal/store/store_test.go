// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package store

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/opencontainers/umoci/oci/layer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testFileName = "some-filename.txt"
)

func TestPutLayer(t *testing.T) {
	for name, tc := range map[string]struct {
		mediaType       string
		reader          func(*testing.T) *closeVerifier
		failureExpected bool
	}{
		"uncompressed": {
			mediaType: "application/vnd.oci.image.layer.v1.tar",
			reader:    uncompressedTarball,
		},
		"gzip": {
			mediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
			reader:    gzipTarball,
		},
		"zstd": {
			mediaType: "application/vnd.oci.image.layer.v1.tar+zstd",
			reader:    zstdTarball,
		},
		"unsupported media type": {
			mediaType:       "application/vnd.oci.image.layer.v1.tar+7z",
			reader:          zstdTarball,
			failureExpected: true,
		},
		"wrong media type": {
			mediaType:       "application/vnd.oci.image.layer.v1.tar+gzip",
			reader:          zstdTarball,
			failureExpected: true,
		},
		"broken pipe": {
			mediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
			reader: func(t *testing.T) *closeVerifier {
				inner := gzipTarball(t)
				rc := &closer{Reader: io.LimitReader(inner, 25), inner: inner}
				return &closeVerifier{ReadCloser: rc}
			},
			failureExpected: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			dir := t.TempDir()
			store := &Store{
				Root:     dir,
				Unpacker: layer.UnpackLayer,
			}
			expectedDigest := streamDigest(t, tc.reader(t))
			expectedPath := filepath.Join(dir, expectedDigest.Algorithm, expectedDigest.Hex)

			layer := &stubLayer{
				reader:    tc.reader(t),
				mediaType: tc.mediaType,
				digest:    expectedDigest,
			}

			digest, putLayerErr := store.PutLayer(layer)

			// Regardless of the outcome, the staging directory must be clean after return.
			dirents, err := os.ReadDir(filepath.Join(dir, stagingDirName))
			require.NoError(err)
			assert.Empty(dirents)

			if tc.failureExpected {
				// If an error occurred, the path must not be present after return.
				require.Error(putLayerErr)
				assert.NoDirExists(expectedPath)
				return
			}

			require.NoError(putLayerErr)
			assert.FileExists(filepath.Join(expectedPath, testFileName))
			assert.Equal(expectedDigest, digest)
		})
	}
}

func TestGetDecompressedReader(t *testing.T) {
	expectedDigest := streamDigest(t, uncompressedTarball(t))
	for name, tc := range map[string]struct {
		mediaType string
		reader    *closeVerifier
		wantErr   error
	}{
		"uncompressed": {
			mediaType: "application/vnd.oci.image.layer.v1.tar",
			reader:    uncompressedTarball(t),
		},
		"gzip": {
			mediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
			reader:    gzipTarball(t),
		},
		"zstd": {
			mediaType: "application/vnd.oci.image.layer.v1.tar+zstd",
			reader:    zstdTarball(t),
		},
		"unsupported media type": {
			mediaType: "application/vnd.oci.image.layer.v1.tar+7z",
			wantErr:   errUnsupportedMediaType,
		},
		"wrong media type": {
			mediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
			reader:    zstdTarball(t),
			wantErr:   pgzip.ErrHeader,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			layer := &stubLayer{
				reader:    tc.reader,
				mediaType: tc.mediaType,
			}

			rc, err := getDecompressedReader(layer)
			if tc.wantErr != nil {
				require.ErrorIs(err, tc.wantErr, "error type: %T", err)
				return
			}
			require.NoError(err)

			hash := streamDigest(t, rc)
			require.Equal(expectedDigest, hash)
			require.NoError(rc.Close())

			require.True(tc.reader.closeCalled.Load())
		})
	}
}

func streamDigest(t *testing.T, r io.Reader) gcr.Hash {
	hash := sha256.New()
	_, err := io.Copy(hash, r)
	require.NoError(t, err)
	return gcr.Hash{
		Algorithm: "sha256",
		Hex:       fmt.Sprintf("%x", hash.Sum(nil)),
	}
}

type closeVerifier struct {
	io.ReadCloser

	closeCalled atomic.Bool
}

func (v *closeVerifier) Close() error {
	v.closeCalled.Store(true)
	return v.ReadCloser.Close()
}

type stubLayer struct {
	gcr.Layer

	reader    *closeVerifier
	mediaType string
	digest    gcr.Hash
}

func (s *stubLayer) Compressed() (io.ReadCloser, error) {
	return s.reader, nil
}

func (s *stubLayer) MediaType() (types.MediaType, error) {
	return types.MediaType(s.mediaType), nil
}

func (s *stubLayer) Digest() (gcr.Hash, error) {
	return s.digest, nil
}

func uncompressedTarball(t *testing.T) *closeVerifier {
	t.Helper()
	require := require.New(t)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	content := []byte("some file content")
	hdr := &tar.Header{
		Name: testFileName,
		Mode: 0o600,
		Uid:  os.Geteuid(),
		Gid:  os.Getegid(),
		Size: int64(len(content)),
	}

	require.NoError(tw.WriteHeader(hdr))
	_, err := tw.Write(content)
	require.NoError(err)
	require.NoError(tw.Close())

	return &closeVerifier{ReadCloser: &closer{Reader: &buf, inner: io.NopCloser(&buf)}}
}

// gzipTarball creates a gzipped tarball.
//
// The compression settings used are obscure on purpose, as is the use of compress/gzip instead of pgzip.
func gzipTarball(t *testing.T) *closeVerifier {
	t.Helper()
	require := require.New(t)

	uncompressed := uncompressedTarball(t)
	defer uncompressed.Close()

	var buf bytes.Buffer
	compressed, err := gzip.NewWriterLevel(&buf, gzip.HuffmanOnly)
	require.NoError(err)
	_, err = io.Copy(compressed, uncompressed)
	require.NoError(err)
	require.NoError(compressed.Close())
	return &closeVerifier{ReadCloser: &closer{Reader: &buf, inner: io.NopCloser(&buf)}}
}

func zstdTarball(t *testing.T) *closeVerifier {
	t.Helper()
	require := require.New(t)

	uncompressed := uncompressedTarball(t)
	defer uncompressed.Close()

	var buf bytes.Buffer
	compressed, err := zstd.NewWriter(&buf)
	require.NoError(err)
	_, err = io.Copy(compressed, uncompressed)
	require.NoError(err)
	require.NoError(compressed.Close())
	return &closeVerifier{ReadCloser: &closer{Reader: &buf, inner: io.NopCloser(&buf)}}
}
