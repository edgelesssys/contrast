// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package asset

import (
	"context"
	"fmt"
	"hash"
	"io"
	"net/url"
	"os"
	"slices"

	"github.com/edgelesssys/contrast/nodeinstaller/internal/fileop"
)

// FileFetcher is a Fetcher that retrieves assets from a file.
// It handles the "file" scheme.
type FileFetcher struct {
	copier copier
}

// NewFileFetcher creates a new file fetcher.
func NewFileFetcher() *FileFetcher {
	return &FileFetcher{copier: fileop.NewDefault()}
}

// Fetch retrieves a file from the local filesystem.
func (f *FileFetcher) Fetch(_ context.Context, uri *url.URL, destination string, expectedSum []byte, hasher hash.Hash) (bool, error) {
	if uri.Scheme != "file" {
		return false, fmt.Errorf("file fetcher does not support scheme %s", uri.Scheme)
	}
	sourceFile, err := os.Open(uri.Path)
	if err != nil {
		return false, fmt.Errorf("opening file: %w", err)
	}
	defer sourceFile.Close()

	if _, err := io.Copy(hasher, sourceFile); err != nil {
		return false, fmt.Errorf("hashing file: %w", err)
	}
	if err := sourceFile.Close(); err != nil {
		return false, fmt.Errorf("closing file: %w", err)
	}
	actualSum := hasher.Sum(nil)
	if !slices.Equal(actualSum, expectedSum) {
		return false, fmt.Errorf("file hash mismatch: expected %x, got %x", expectedSum, actualSum)
	}
	changed, err := f.copier.CopyOnDiff(uri.Path, destination)
	if err != nil {
		return false, fmt.Errorf("copying file: %w", err)
	}
	return changed, nil
}

// FetchUnchecked retrieves a file from the local filesystem without verifying its integrity.
func (f *FileFetcher) FetchUnchecked(_ context.Context, uri *url.URL, destination string) (bool, error) {
	if uri.Scheme != "file" {
		return false, fmt.Errorf("file fetcher does not support scheme %s", uri.Scheme)
	}
	return f.copier.CopyOnDiff(uri.Path, destination)
}

type copier interface {
	CopyOnDiff(src, dst string) (bool, error)
}
