package snp

import (
	"log/slog"

	"github.com/edgelesssys/nunki/internal/memstore"
	"github.com/google/go-sev-guest/verify/trust"
)

type cachedKDSHTTPClient struct {
	trust.HTTPSGetter
	logger *slog.Logger

	cache *memstore.Store[string, cacheEntry]
}

func newCachedKDSHTTPClient(log *slog.Logger) *cachedKDSHTTPClient {
	trust.DefaultHTTPSGetter()
	return &cachedKDSHTTPClient{
		HTTPSGetter: trust.DefaultHTTPSGetter(),
		logger:      log.WithGroup("cached-kds-http-client"),
		cache:       memstore.New[string, cacheEntry](),
	}
}

func (c *cachedKDSHTTPClient) Get(url string) ([]byte, error) {
	if cached, ok := c.cache.Get(url); ok {
		c.logger.Debug("Get cached", "url", url)
		return cached.data, nil
	}

	c.logger.Debug("Get not cached", "url", url)
	res, err := c.HTTPSGetter.Get(url)
	if err != nil {
		return nil, err
	}
	c.cache.Set(url, cacheEntry{
		data: res,
	})
	return res, nil
}

type cacheEntry struct {
	data []byte
}
