package snp

import (
	"log/slog"

	"github.com/edgelesssys/nunki/internal/memstore"
	"github.com/google/go-sev-guest/verify/trust"
	"k8s.io/utils/clock"
)

type cachedKDSHTTPClient struct {
	trust.HTTPSGetter
	logger *slog.Logger

	gcTicker clock.Ticker
	cache    *memstore.Store[string, []byte]
}

func newCachedKDSHTTPClient(ticker clock.Ticker, log *slog.Logger) *cachedKDSHTTPClient {
	trust.DefaultHTTPSGetter()

	c := &cachedKDSHTTPClient{
		HTTPSGetter: trust.DefaultHTTPSGetter(),
		logger:      log.WithGroup("cached-kds-http-client"),
		cache:       memstore.New[string, []byte](),
		gcTicker:    ticker,
	}

	return c
}

func (c *cachedKDSHTTPClient) Get(url string) ([]byte, error) {
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
