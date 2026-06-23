// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package upstream

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/sync/singleflight"
)

// Result is a captured upstream response.
type Result struct {
	Status int
	Header http.Header
	Body   []byte
}

// Fetcher pulls upstream responses.
type Fetcher struct {
	client *http.Client
	group  singleflight.Group
}

// New returns a Fetcher backed by client.
func New(client *http.Client) *Fetcher {
	return &Fetcher{client: client}
}

// Get fetches the given url.
func (f *Fetcher) Get(ctx context.Context, url string) (*Result, error) {
	v, err, _ := f.group.Do(url, func() (any, error) {
		return f.doGet(ctx, url)
	})
	if err != nil {
		return nil, err
	}
	res, ok := v.(*Result)
	if !ok {
		return nil, fmt.Errorf("unexpected singleflight result type %T", v)
	}
	return res, nil
}

func (f *Fetcher) doGet(ctx context.Context, url string) (*Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream fetch: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading upstream body: %w", err)
	}
	return &Result{Status: resp.StatusCode, Header: resp.Header, Body: body}, nil
}
