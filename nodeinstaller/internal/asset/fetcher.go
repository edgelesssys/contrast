// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package asset

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"net/url"
)

// Fetcher can retrieve assets from various sources.
// It works by delegating to a handler for the scheme of the source URI.
type Fetcher struct {
	handlers map[string]handler
}

// NewDefaultFetcher creates a new fetcher with default handlers.
func NewDefaultFetcher() *Fetcher {
	fileFetcher := NewFileFetcher()
	httpFetcher := NewHTTPFetcher()
	return NewFetcher(map[string]handler{
		"file":  fileFetcher,
		"http":  httpFetcher,
		"https": httpFetcher,
	})
}

// NewFetcher creates a new fetcher.
func NewFetcher(handlers map[string]handler) *Fetcher {
	return &Fetcher{
		handlers: handlers,
	}
}

// Fetch retrieves a file from a source URI.
func (f *Fetcher) Fetch(ctx context.Context, sourceURI, destination, integrity string) (changed bool, retErr error) {
	uri, err := url.Parse(sourceURI)
	if err != nil {
		return false, err
	}
	hasher, expectedSum, err := hashFromIntegrity(integrity)
	if err != nil {
		return false, err
	}
	schemeFetcher := f.handlers[uri.Scheme]
	if schemeFetcher == nil {
		return false, fmt.Errorf("no handler for scheme %s", uri.Scheme)
	}
	return schemeFetcher.Fetch(ctx, uri, destination, expectedSum, hasher)
}

// FetchUnchecked retrieves a file from a source URI without verifying its integrity.
func (f *Fetcher) FetchUnchecked(ctx context.Context, sourceURI, destination string) (changed bool, retErr error) {
	uri, err := url.Parse(sourceURI)
	if err != nil {
		return false, err
	}
	schemeFetcher := f.handlers[uri.Scheme]
	if schemeFetcher == nil {
		return false, fmt.Errorf("no handler for scheme %s", uri.Scheme)
	}
	return schemeFetcher.FetchUnchecked(ctx, uri, destination)
}

type handler interface {
	Fetch(ctx context.Context, uri *url.URL, destination string, expectedSum []byte, hasher hash.Hash) (bool, error)
	FetchUnchecked(ctx context.Context, uri *url.URL, destination string) (bool, error)
}

func hashFromIntegrity(integrity string) (hash.Hash, []byte, error) {
	var hash hash.Hash
	switch integrity[:7] {
	case "sha256-":
		hash = sha256.New()
	case "sha384-":
		hash = sha512.New384()
	case "sha512-":
		hash = sha512.New()
	default:
		return nil, nil, fmt.Errorf("unsupported hash algorithm: %s", integrity[:7])
	}
	expectedSum, err := base64.StdEncoding.DecodeString(integrity[7:])
	if err != nil {
		return nil, nil, fmt.Errorf("decoding integrity value: %w", err)
	}
	return hash, expectedSum, nil
}
