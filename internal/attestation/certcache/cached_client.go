// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package certcache

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-tdx-guest/pcs"
	"github.com/google/go-tdx-guest/verify/trust"
	"k8s.io/utils/clock"
	testingclock "k8s.io/utils/clock/testing"
)

var (
	snpCrlPath     = regexp.MustCompile(`^/vcek/v1/[A-Za-z]*/crl$`)
	tdxRootCrlPath = regexp.MustCompile(`^/IntelSGXRootCA\.der$`)
	tdxBasePath    = regexp.MustCompile(`^/(sgx|tdx)/certification/v4`)
)

func alwaysRevalidate(rawURL string) bool {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return false
	}
	return snpCrlPath.MatchString(u.Path) || tdxRootCrlPath.MatchString(u.Path) || tdxBasePath.MatchString(u.Path)
}

var collateralProxyBase string

const intelCertificatesHost = "certificates.trustedservices.intel.com"

// SetCollateralProxy routes attestation-collateral fetches through the in-cluster collateral-proxy at proxyURL.
func SetCollateralProxy(proxyURL string) {
	proxyURL = strings.TrimRight(proxyURL, "/")
	if proxyURL == "" {
		return
	}
	kds.KDSBaseURL = proxyURL
	pcs.SgxBaseURL = proxyURL + "/sgx/certification/v4"
	pcs.TdxBaseURL = proxyURL + "/tdx/certification/v4"
	collateralProxyBase = proxyURL
}

func redirectToProxy(rawURL string) string {
	if collateralProxyBase == "" {
		return rawURL
	}
	u, err := neturl.Parse(rawURL)
	if err != nil || u.Host != intelCertificatesHost {
		return rawURL
	}
	redirected := collateralProxyBase + u.Path
	if u.RawQuery != "" {
		redirected += "?" + u.RawQuery
	}
	return redirected
}

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
		ContextHTTPSGetter: NewRetryHTTPSGetter(http.DefaultClient, 5*time.Second),
		logger:             log,
		cache:              s,
		gcTicker:           ticker,
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

	url = redirectToProxy(url)
	log := c.logger.With("url", url)

	if alwaysRevalidate(url) {
		// For CRLs or TDX TCB/QeIdentity always query. When request failure, fallback to cache.
		log.Debug("Requesting URL")
		header, body, err := c.ContextHTTPSGetter.GetContext(ctx, url)
		if err == nil {
			if data, err := json.Marshal(cacheEntry{header, body}); err == nil {
				c.cache.Set(url, data)
			} else {
				log.Warn("Failed to marshal response, not writing to cache", "error", err)
			}
			return header, body, nil
		}
		log.Warn("Failed requesting URL from KDS/PCS", "error", err)
		if cached, ok := c.cache.Get(url); ok {
			var entry cacheEntry
			if err := json.Unmarshal(cached, &entry); err == nil {
				log.Warn("Falling back to cached entry")
				return entry.Header, entry.Body, nil
			}
		}
		log.Warn("Entry not found in cache")
		return nil, nil, err
	}
	// For VCEK get cache first and request if not present
	if cached, ok := c.cache.Get(url); ok {
		var entry cacheEntry
		if err := json.Unmarshal(cached, &entry); err == nil {
			log.Debug("Cache hit")
			return entry.Header, entry.Body, nil
		}
	}
	log.Debug("Cache miss, requesting")
	header, body, err := c.ContextHTTPSGetter.GetContext(ctx, url)
	if err != nil {
		return nil, nil, err
	}
	if data, err := json.Marshal(cacheEntry{header, body}); err == nil {
		c.cache.Set(url, data)
	} else {
		log.Warn("Failed to marshal response, not writing to cache", "error", err)
	}
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
