// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package certcache

import (
	"context"
	"log/slog"
	"regexp"
	"time"

	"github.com/google/go-sev-guest/verify/trust"
	"k8s.io/utils/clock"
	testingclock "k8s.io/utils/clock/testing"
)

var crlURL = regexp.MustCompile(`^https://kdsintf\.amd\.com/vcek/v1/[A-Za-z]*/crl$`)

// CachedHTTPSGetter is a HTTPS client that caches responses in memory.
type CachedHTTPSGetter struct {
	trust.ContextHTTPSGetter
	logger *slog.Logger

	gcTicker clock.Ticker
	cache    store
}

// NewCachedHTTPSGetter returns a new CachedHTTPSGetter.
func NewCachedHTTPSGetter(s store, ticker clock.Ticker, log *slog.Logger) *CachedHTTPSGetter {
	c := &CachedHTTPSGetter{
		ContextHTTPSGetter: &trust.RetryHTTPSGetter{
			// Default values taken from trust.DefaultHTTPSGetter.
			Timeout:       2 * time.Minute,
			MaxRetryDelay: 30 * time.Second,
			Getter:        &trust.SimpleHTTPSGetter{},
		},
		logger:   log,
		cache:    s,
		gcTicker: ticker,
	}
	return c
}

// Get makes a GET request to the given URL.
func (c *CachedHTTPSGetter) Get(url string) ([]byte, error) {
	return c.GetContext(context.Background(), url)
}

// GetContext makes a GET request to the given URL using the given context.
func (c *CachedHTTPSGetter) GetContext(ctx context.Context, url string) ([]byte, error) {
	select {
	case <-c.gcTicker.C():
		c.logger.Debug("Garbage collecting")
		c.cache.Clear()
	default:
	}

	if crlURL.MatchString(url) {
		// For CRLs always query. When request failure, fallback to cache.
		c.logger.Debug("Request CRL", "url", url)
		res, err := c.ContextHTTPSGetter.GetContext(ctx, url)
		if err == nil {
			c.cache.Set(url, res)
			return res, nil
		}
		c.logger.Warn("Failed requesting CRL from KDS", "error", err)
		if cached, ok := c.cache.Get(url); ok {
			c.logger.Warn("Falling back to cached CRL", "url", url)
			return cached, nil
		}
		c.logger.Warn("CRL not found in cache", "url", url)
		return nil, err
	}
	// For VCEK get cache first and request if not present
	if cached, ok := c.cache.Get(url); ok {
		c.logger.Debug("Cache hit", "url", url)
		return cached, nil
	}
	c.logger.Debug("Cache miss, requesting", "url", url)
	res, err := c.ContextHTTPSGetter.GetContext(ctx, url)
	if err != nil {
		return nil, err
	}
	c.cache.Set(url, res)
	return res, nil
}

// NeverGCTicker is a ticker that never ticks.
var NeverGCTicker = testingclock.NewFakeClock(time.Now()).NewTicker(0)

type store interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
	Clear()
}
