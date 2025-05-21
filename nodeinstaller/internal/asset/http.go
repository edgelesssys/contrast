// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package asset

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/edgelesssys/contrast/nodeinstaller/internal/fileop"
)

// HTTPFetcher is a Fetcher that retrieves assets from http(s).
// It handles the "http" and "https" schemes.
type HTTPFetcher struct {
	mover  mover
	client *http.Client
}

// NewHTTPFetcher creates a new HTTP fetcher.
func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{mover: fileop.NewDefault(), client: http.DefaultClient}
}

// Fetch retrieves a file from an HTTP server.
func (f *HTTPFetcher) Fetch(ctx context.Context, uri *url.URL, destination string, expectedSum []byte, hasher hash.Hash) (bool, error) {
	if uri.Scheme != "http" && uri.Scheme != "https" {
		return false, fmt.Errorf("http fetcher does not support scheme %s", uri.Scheme)
	}

	if existing, err := os.Open(destination); err == nil {
		defer existing.Close()
		if _, err := io.Copy(hasher, existing); err != nil {
			return false, fmt.Errorf("hashing existing file %s: %w", destination, err)
		}
		if sum := hasher.Sum(nil); bytes.Equal(sum, expectedSum) {
			// File already exists and has the correct hash
			return false, nil
		}
		hasher.Reset()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), http.NoBody)
	if err != nil {
		return false, fmt.Errorf("creating request: %w", err)
	}
	response, err := f.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("fetching file: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return false, fmt.Errorf("fetching file: %s", response.Status)
	}

	defer response.Body.Close()

	tmpfile, err := os.CreateTemp("", "download")
	if err != nil {
		return false, err
	}
	defer tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	reader := io.TeeReader(response.Body, hasher)

	if _, err := io.Copy(tmpfile, reader); err != nil {
		return false, fmt.Errorf("downloading file contents from %s: %w", uri.String(), err)
	}

	sum := hasher.Sum(nil)
	if !bytes.Equal(sum, expectedSum) {
		return false, fmt.Errorf("hash mismatch for %s: expected %x, got %x", uri.String(), expectedSum, sum)
	}
	if err := tmpfile.Sync(); err != nil {
		return false, fmt.Errorf("syncing file %s: %w", tmpfile.Name(), err)
	}
	if err := f.mover.Move(tmpfile.Name(), destination); err != nil {
		return false, fmt.Errorf("moving file: %w", err)
	}

	return true, nil
}

// FetchUnchecked retrieves a file from an HTTP server without verifying its integrity.
func (f *HTTPFetcher) FetchUnchecked(ctx context.Context, uri *url.URL, destination string) (bool, error) {
	if uri.Scheme != "http" && uri.Scheme != "https" {
		return false, fmt.Errorf("http fetcher does not support scheme %s", uri.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), http.NoBody)
	if err != nil {
		return false, fmt.Errorf("creating request: %w", err)
	}
	response, err := f.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("fetching file: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return false, fmt.Errorf("fetching file: %s", response.Status)
	}

	dstFile, err := os.Create(destination)
	if err != nil {
		return false, err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, response.Body)
	if err != nil {
		return false, fmt.Errorf("downloading file contents from %s: %w", uri.String(), err)
	}
	return true, nil
}

type mover interface {
	Move(src, dst string) error
}
