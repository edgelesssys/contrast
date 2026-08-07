// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

// Package transport implements the HTTP plumbing shared by the SDK's API versions.
//
// It's internal so that the versioned API packages can share one connection setup
// without exposing it as part of the SDK's public surface.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/edgelesssys/contrast/apitypes"
)

// ErrBaseURLUnset is returned when a request is attempted without a base URL.
var ErrBaseURLUnset = errors.New("no base URL set, use WithBaseURL")

// Client performs JSON requests against the Coordinator's HTTP API.
type Client struct {
	// HTTPClient is used to contact the Coordinator.
	HTTPClient *http.Client
	// BaseURL is the Coordinator's HTTP API root, e.g. "http://coordinator:1314".
	BaseURL string
	// Log receives diagnostics that aren't part of a returned error.
	Log *slog.Logger
}

// URL resolves an API path like "/v1/manifest" against the client's base URL.
func (c *Client) URL(path string) (string, error) {
	if c.BaseURL == "" {
		return "", ErrBaseURLUnset
	}
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", fmt.Errorf("parsing base URL %q: %w", c.BaseURL, err)
	}
	ref, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("parsing path %q: %w", path, err)
	}
	// Ensure a base URL with a path prefix, as used by reverse proxies, keeps that
	// prefix: "https://proxy/contrast" + "/v1/manifest" resolves to
	// "https://proxy/contrast/v1/manifest".
	base.Path = strings.TrimSuffix(base.Path, "/") + ref.EscapedPath()
	base.RawQuery = ref.RawQuery
	return base.String(), nil
}

// DoJSON sends reqBody JSON-encoded to the given API path and returns the raw response body.
//
// If the Coordinator responds with a non-OK status, the body is parsed as an
// [apitypes.APIError] and returned as an error.
func (c *Client) DoJSON(ctx context.Context, method, path string, reqBody any) ([]byte, error) {
	url, err := c.URL(path)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if reqBody != nil {
		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("creating request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("constructing HTTP request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	httpResp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		errBody, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response (status code %d): %w", httpResp.StatusCode, err)
		}
		details := httpResp.Status
		var apiErr apitypes.APIError
		if err := json.Unmarshal(errBody, &apiErr); err == nil {
			details = apiErr.Err
		} else {
			c.Log.Error("parsing error response", "err", err, "response", string(errBody))
		}
		return nil, fmt.Errorf("HTTP API call failed with %d (%s): %s", httpResp.StatusCode, http.StatusText(httpResp.StatusCode), details)
	}

	resp, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading HTTP response body: %w", err)
	}
	return resp, nil
}
