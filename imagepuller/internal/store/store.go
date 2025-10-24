package store

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/umoci/oci/layer"
)

type Store struct {
	Root    string
	Staging string
}

func (s *Store) PutLayer(what io.Reader, expectedDigest string) (retErr error) {
	algo, digest, ok := strings.Cut(expectedDigest, ":")
	if !ok {
		return fmt.Errorf("digest should contain colon: %q", expectedDigest)
	}
	algoPath := filepath.Join(s.Root, algo)
	if err := os.MkdirAll(algoPath, 0o755); err != nil {
		return fmt.Errorf("creating layer dir %q: %w", algoPath, err)
	}
	targetPath := filepath.Join(algoPath, digest)
	if _, err := os.Stat(targetPath); err == nil {
		// Nothing to do, the layer is already pulled.
		return nil
	}
	tempdir, err := os.MkdirTemp(s.Staging, expectedDigest)
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tempdir)

	// TODO(burgerdev): take prefix of expect to create hasher
	hash := sha256.New()

	hashingReader := io.TeeReader(what, hash)

	// TODO(burgerdev): inspect media type to determine compressor
	gzipReader, err := gzip.NewReader(hashingReader)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}

	opts := &layer.UnpackOptions{
		OnDiskFormat: layer.OverlayfsRootfs{},
	}
	if err := layer.UnpackLayer(tempdir, gzipReader, opts); err != nil {
		return fmt.Errorf("unpacking layer: %w", err)
	}

	// Apparently some tar/gzip implementations add padding.
	// Make sure that all bytes are hashed, even if not consumed by UnpackLayer.
	if _, err := io.Copy(io.Discard, hashingReader); err != nil {
		return fmt.Errorf("reading trailing bytes: %w", err)
	}

	got := fmt.Sprintf("sha256:%x", hash.Sum(nil))
	if expectedDigest != got {
		return fmt.Errorf("comparing digests: got %q, want %q", got, expectedDigest)
	}
	if err := os.Rename(tempdir, targetPath); err != nil {
		return fmt.Errorf("moving unpacked layer: %w", err)
	}
	return nil
}
