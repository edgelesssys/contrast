package snp

import (
	"log/slog"

	"github.com/edgelesssys/nunki/internal/logger"
	"github.com/google/go-sev-guest/verify/trust"
	"k8s.io/utils/clock"
)

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
		logger:      slog.New(logger.NewHandler(log.Handler(), "cached-kds-http-client")),
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

	if cached, ok := c.cache.Get(url); ok {
		c.logger.Debug("Get cached", "url", url)
		return cached, nil
	}

	c.logger.Debug("Get not cached", "url", url)
	res, err := c.HTTPSGetter.Get(url)
	if err != nil {
		return nil, err
	}
	c.cache.Set(url, res)
	return res, nil
}

type store interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
	Clear()
}
