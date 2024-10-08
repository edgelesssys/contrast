// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package certcache

import (
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
	trust.HTTPSGetter
	logger *slog.Logger

	gcTicker clock.Ticker
	cache    store
}

// NewCachedHTTPSGetter returns a new CachedHTTPSGetter.
func NewCachedHTTPSGetter(s store, ticker clock.Ticker, log *slog.Logger) *CachedHTTPSGetter {
	c := &CachedHTTPSGetter{
		HTTPSGetter: trust.DefaultHTTPSGetter(),
		logger:      log,
		cache:       s,
		gcTicker:    ticker,
	}
	return c
}

// Get makes a GET request to the given URL.
func (c *CachedHTTPSGetter) Get(url string) ([]byte, error) {
	select {
	case <-c.gcTicker.C():
		c.logger.Debug("Garbage collecting")
		c.cache.Clear()
	default:
	}

	// Don't cache CRLs. Unlike VCEKs, these can change over time and the KDS
	// doesn't rate-limit requests to these.
	canCache := !crlURL.MatchString(url)

	if canCache {
		if cached, ok := c.cache.Get(url); ok {
			c.logger.Debug("Get cached", "url", url)
			return cached, nil
		}
	}

	c.logger.Debug("Get not cached", "url", url)
	res, err := c.HTTPSGetter.Get(url)
	if err != nil {
		return nil, err
	}
	if canCache {
		c.cache.Set(url, res)
	}
	return res, nil
}

// NeverGCTicker is a ticker that never ticks.
var NeverGCTicker = testingclock.NewFakeClock(time.Now()).NewTicker(0)

type store interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
	Clear()
}
