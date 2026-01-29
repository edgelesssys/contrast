// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package certcache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-tdx-guest/verify/trust"
	"github.com/stretchr/testify/assert"
	testingclock "k8s.io/utils/clock/testing"
)

func TestRetryHTTPSGetter(t *testing.T) {
	for name, tc := range map[string]struct {
		numFailures int
		cancelAfter time.Duration
		statusCode  int
		wantErr     error
		wantHeader  map[string][]string
		wantBody    []byte
	}{
		"no failures": {},
		"headers and body": {
			wantHeader: map[string][]string{
				"Foo": {"bar", "baz"},
				"Qux": {"x", "y", "z"},
			},
			wantBody: []byte("Hello world!"),
		},
		"one failure no retries": {
			numFailures: 1,
			cancelAfter: time.Second,
			statusCode:  http.StatusBadGateway,
			wantErr:     context.Canceled,
		},
		"one failure with retries": {
			numFailures: 1,
			statusCode:  http.StatusBadGateway,
			wantHeader: map[string][]string{
				"Foo": {"bar", "baz"},
				"Qux": {"x", "y", "z"},
			},
			wantBody: []byte("Hello world!"),
		},
		"5 failures 4 retries": {
			numFailures: 5,
			cancelAfter: 4 * time.Second,
			statusCode:  http.StatusBadGateway,
			wantErr:     context.Canceled,
		},
		"5 failures with retries": {
			numFailures: 5,
			statusCode:  http.StatusBadGateway,
			wantHeader: map[string][]string{
				"Foo": {"bar", "baz"},
				"Qux": {"x", "y", "z"},
			},
			wantBody: []byte("Hello world!"),
		},
		"bad request": {
			numFailures: 1,
			statusCode:  http.StatusBadRequest,
			wantErr:     &httpError{code: http.StatusBadRequest},
		},
		"gone": {
			numFailures: 1,
			statusCode:  http.StatusGone,
			wantErr:     &httpError{code: http.StatusGone},
		},
		"partial content": {
			numFailures: 1,
			statusCode:  http.StatusPartialContent,
			wantErr:     &httpError{code: http.StatusPartialContent},
		},
		"switching protocols": {
			numFailures: 1,
			statusCode:  http.StatusSwitchingProtocols,
			wantErr:     &httpError{code: http.StatusSwitchingProtocols},
		},
		"non-authoritative info": {
			numFailures: 1,
			statusCode:  http.StatusNonAuthoritativeInfo,
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			t.Cleanup(cancel)

			clock := &testingclock.FakeClock{}

			handler := &stubHandler{
				clock:       clock,
				numFailures: tc.numFailures,
				statusCode:  tc.statusCode,
				headers:     tc.wantHeader,
				body:        tc.wantBody,
			}

			srv := httptest.NewServer(handler)
			t.Cleanup(srv.Close)

			getter := &RetryHTTPSGetter{
				client:   srv.Client(),
				interval: time.Second,
				clock:    clock,
			}

			wg := sync.WaitGroup{}
			wg.Go(func() {
				if tc.cancelAfter == 0 {
					return
				}
				select {
				case <-clock.After(tc.cancelAfter):
				case <-ctx.Done():
				}
				cancel()
			})
			wg.Go(func() {
				assert := assert.New(t)
				headers, body, err := getter.GetContext(ctx, srv.URL)

				if tc.wantErr != nil {
					assert.ErrorIs(err, tc.wantErr)
					assert.Nil(headers)
					assert.Nil(body)
					return
				}

				assert.NotNil(headers)
				for key, values := range tc.wantHeader {
					if assert.Contains(headers, key) {
						assert.Equal(values, headers[key])
					}
				}
				assert.Equal(tc.wantBody, body)
				assert.NoError(err)
			})

			wg.Wait()
		})
	}
}

// Test that the expected interface is implemented.
var _ = trust.ContextHTTPSGetter(&RetryHTTPSGetter{})

type stubHandler struct {
	requestCount atomic.Int32
	clock        *testingclock.FakeClock
	numFailures  int
	statusCode   int
	headers      map[string][]string
	body         []byte
}

func (h *stubHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	defer h.clock.Step(time.Second)
	if h.requestCount.Add(1) <= int32(h.numFailures) {
		w.WriteHeader(h.statusCode)
		return
	}
	for key, values := range h.headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	_, _ = w.Write(h.body)
}
