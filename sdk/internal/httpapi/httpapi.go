// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

// Package httpapi implements the HTTP request plumbing shared by the SDK's API versions.
//
// The versioned API packages, like sdk/apiv1, can't import the SDK itself, because the SDK imports them.
// This package holds the request plumbing they'd otherwise have to duplicate.
// It's internal so that this arrangement doesn't leak into the SDK's public surface.
package httpapi

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

// resolveURL resolves an API path like "/v1/manifest" against the client's base URL.
func (c *Client) resolveURL(path string) (string, error) {
	if c.BaseURL == "" {
		return "", ErrBaseURLUnset
	}
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", fmt.Errorf("parsing base URL %q: %w", c.BaseURL, err)
	}
	if (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return "", fmt.Errorf("base URL %q: expected http(s)://host[:port][/prefix], set it with WithBaseURL", c.BaseURL)
	}
	ref, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("parsing path %q: %w", path, err)
	}
	// Ensure a base URL with a path prefix, as used by reverse proxies, keeps that
	// prefix: "https://proxy/contrast" + "/v1/manifest" resolves to
	// "https://proxy/contrast/v1/manifest".
	joined := base.JoinPath(ref.EscapedPath())
	joined.RawQuery = ref.RawQuery
	return joined.String(), nil
}

// DoJSON sends reqBody JSON-encoded to the given API path and returns the raw response body.
//
// If the Coordinator responds with a non-OK status, the body is parsed into an [apitypes.APIError].
func (c *Client) DoJSON(ctx context.Context, method, path string, reqBody any) ([]byte, error) {
	reqURL, err := c.resolveURL(path)
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

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
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
		apiErr := &apitypes.APIError{Err: httpResp.Status}
		if err := json.Unmarshal(errBody, apiErr); err != nil {
			c.Log.Error("parsing error response", "err", err, "response", string(errBody))
			apiErr.Err = httpResp.Status
		} else if apiErr.Err == "" {
			apiErr.Err = httpResp.Status
		}
		apiErr.StatusCode = httpResp.StatusCode
		return nil, apiErr
	}

	resp, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading HTTP response body: %w", err)
	}
	return resp, nil
}
