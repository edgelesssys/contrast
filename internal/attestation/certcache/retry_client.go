// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package certcache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/edgelesssys/contrast/internal/retry"
	"k8s.io/utils/clock"
)

// RetryHTTPSGetter is an improved version of go-tdx-guest's trust.RetryHTTPSGetter that takes HTTP
// semantics into account.
//
// Instances must be created with NewRetryHTTPSGetter.
type RetryHTTPSGetter struct {
	client   *http.Client
	interval time.Duration

	clock clock.WithTicker
}

// NewRetryHTTPSGetter constructs a new RetryHTTPSGetter.
//
// The getter will use the given client to do individual requests, and the requests will be spaced
// with the given interval.
func NewRetryHTTPSGetter(client *http.Client, interval time.Duration) *RetryHTTPSGetter {
	return &RetryHTTPSGetter{
		client:   client,
		interval: interval,
		clock:    clock.RealClock{},
	}
}

// GetContext fetches the HTTP headers and body with a GET request to the given URL.
//
// GetContext retries until it gets a successful response, or a client-side HTTP error (4XX).
func (g *RetryHTTPSGetter) GetContext(ctx context.Context, url string) (headers map[string][]string, body []byte, retErr error) {
	doer := retry.DoerFunc(func(ctx context.Context) error {
		var err error
		headers, body, err = g.fetchOnce(ctx, url)
		return err
	})

	retrier := retry.NewIntervalRetrier(doer, g.interval, shouldRetry, g.clock)
	if err := retrier.Do(ctx); err != nil {
		return nil, nil, err
	}
	return headers, body, nil
}

func (g *RetryHTTPSGetter) fetchOnce(ctx context.Context, url string) (http.Header, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, g.interval)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("creating HTTP request object: %w", err)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("requesting URL %q: %w", url, err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, nil, fmt.Errorf("reading HTTP response: %w", err)
	}

	// As per the HTTP spec, error codes are 4XX and 5xx. However, the other codes are essentially
	// errors for our use case, too:
	// - 1XX codes usually don't have a response body (at least not the one we're looking for).
	// - 2XX (except the ones checked) also don't, at least for GET requests.
	// - 3XX should be handled internally by http.Client.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNonAuthoritativeInfo {
		return nil, nil, &httpError{code: resp.StatusCode, status: resp.Status}
	}
	return resp.Header, buf.Bytes(), nil
}

// shouldRetry determines whether an HTTP request performed by fetchOnce should be retried.
//
// The retry logic is:
//   - If the HTTP request failed at the transport layer, it should be retried.
//   - If the server responded with a server error [1], the request should be retried.
//   - Otherwise, it should not be retried.
//
// [1]: https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status#server_error_responses
func shouldRetry(err error) bool {
	var httpError *httpError
	if !errors.As(err, &httpError) {
		// TODO(burgerdev): this could be a bit more sensitive, but it's probably better to retry than not to.
		return true
	}
	return httpError.code >= 500
}

type httpError struct {
	code   int
	status string
}

func (err *httpError) Error() string {
	return fmt.Sprintf("HTTP request returned status %d (%q)", err.code, err.status)
}

func (err *httpError) Is(target error) bool {
	targetHTTPError, ok := target.(*httpError)
	if !ok {
		return false
	}
	return targetHTTPError.code == err.code
}
