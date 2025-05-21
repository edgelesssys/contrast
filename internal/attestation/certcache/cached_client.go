// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package certcache

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"time"

	"github.com/google/go-tdx-guest/verify/trust"
	"k8s.io/utils/clock"
	testingclock "k8s.io/utils/clock/testing"
)

var (
	snpCrlURL = regexp.MustCompile(`^https://kdsintf\.amd\.com/vcek/v1/[A-Za-z]*/crl$`)
	tdxCrlURL = regexp.MustCompile(`^https://api\.trustedservices\.intel\.com/sgx/certification/v4/pckcrl\?ca=(platform|processor)&encoding=der$`)
)

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
func (c *CachedHTTPSGetter) Get(url string) (map[string][]string, []byte, error) {
	return c.GetContext(context.Background(), url)
}

// GetContext makes a GET request to the given URL using the given context.
func (c *CachedHTTPSGetter) GetContext(ctx context.Context, url string) (map[string][]string, []byte, error) {
	select {
	case <-c.gcTicker.C():
		c.logger.Debug("Garbage collecting")
		c.cache.Clear()
	default:
	}

	if snpCrlURL.MatchString(url) || tdxCrlURL.MatchString(url) {
		// For CRLs always query. When request failure, fallback to cache.
		c.logger.Debug("Request CRL", "url", url)
		header, body, err := c.ContextHTTPSGetter.GetContext(ctx, url)
		if err == nil {
			data, err := json.Marshal(cacheEntry{header, body})
			if err != nil {
				c.logger.Error("Failed to marshal CRL response", "error", err)
				return nil, nil, err
			}
			c.cache.Set(url, data)
			return header, body, nil
		}
		c.logger.Warn("Failed requesting CRL from KDS/PCS", "error", err)
		if cached, ok := c.cache.Get(url); ok {
			c.logger.Warn("Falling back to cached CRL", "url", url)
			var entry cacheEntry
			if err := json.Unmarshal(cached, &entry); err != nil {
				c.logger.Error("Failed to unmarshal cached CRL", "error", err)
				return nil, nil, err
			}
			return entry.Header, entry.Body, nil
		}
		c.logger.Warn("CRL not found in cache", "url", url)
		return nil, nil, err
	}
	// For VCEK get cache first and request if not present
	if cached, ok := c.cache.Get(url); ok {
		c.logger.Debug("Cache hit", "url", url)
		var entry cacheEntry
		if err := json.Unmarshal(cached, &entry); err != nil {
			c.logger.Error("Failed to unmarshal cached response", "error", err)
			return nil, nil, err
		}
		return entry.Header, entry.Body, nil
	}
	c.logger.Debug("Cache miss, requesting", "url", url)
	header, body, err := c.ContextHTTPSGetter.GetContext(ctx, url)
	if err != nil {
		return nil, nil, err
	}
	data, err := json.Marshal(cacheEntry{header, body})
	if err != nil {
		c.logger.Error("Failed to marshal response", "error", err)
		return nil, nil, err
	}
	c.cache.Set(url, data)
	return header, body, nil
}

// CachedHTTPSGetterSNP is a HTTPS client that caches responses in memory for SNP.
type CachedHTTPSGetterSNP struct {
	*CachedHTTPSGetter
}

// SNPGetter returns a CachedHTTPSGetterSNP.
func (c *CachedHTTPSGetter) SNPGetter() *CachedHTTPSGetterSNP {
	return &CachedHTTPSGetterSNP{c}
}

// Get makes a GET request to the given URL.
func (c *CachedHTTPSGetterSNP) Get(url string) ([]byte, error) {
	_, body, err := c.CachedHTTPSGetter.Get(url)
	return body, err
}

// GetContext makes a GET request to the given URL using the given context.
func (c *CachedHTTPSGetterSNP) GetContext(ctx context.Context, url string) ([]byte, error) {
	_, body, err := c.CachedHTTPSGetter.GetContext(ctx, url)
	return body, err
}

type cacheEntry struct {
	Header map[string][]string
	Body   []byte
}

// NeverGCTicker is a ticker that never ticks.
var NeverGCTicker = testingclock.NewFakeClock(time.Now()).NewTicker(0)

type store interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
	Clear()
}
