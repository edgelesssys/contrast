// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cache

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultTTL = time.Hour

// Entry is a cached HTTP response.
type Entry struct {
	URL        string      `json:"url"`
	Status     int         `json:"status"`
	Header     http.Header `json:"header"`
	Body       []byte      `json:"body"`
	FreshUntil time.Time   `json:"freshUntil"`
}

// Cache is an in-memory and on-disk cache for upstream HTTP responses.
type Cache struct {
	dir     string
	mu      sync.RWMutex
	entries map[string]*Entry
}

// New opens or creates a cache at dir, loading existing entries into memory.
func New(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	c := &Cache{dir: dir, entries: map[string]*Entry{}}
	if err := c.loadAll(); err != nil {
		return nil, err
	}
	return c, nil
}

// Get returns the entry for url and whether or not it is fresh.
func (c *Cache) Get(url string) (*Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[url]
	if !ok {
		return nil, false
	}
	return e, time.Now().Before(e.FreshUntil)
}

// Put stores a response under url, computing freshness from headers / body.
func (c *Cache) Put(url string, status int, header http.Header, body []byte) (*Entry, error) {
	e := &Entry{
		URL:        url,
		Status:     status,
		Header:     header.Clone(),
		Body:       body,
		FreshUntil: time.Now().Add(freshness(status, header, body)),
	}
	// Hold the lock across the disk write and the in-memory update so concurrent
	// Puts can't end up with the disk and the map reflecting different writers.
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.writeDisk(e); err != nil {
		return nil, err
	}
	c.entries[url] = e
	return e, nil
}

func (c *Cache) loadAll() error {
	return filepath.WalkDir(c.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		var e Entry
		if err := json.NewDecoder(f).Decode(&e); err != nil {
			slog.Warn("skipping corrupt cache entry", "path", path, "err", err)
			return nil
		}
		c.entries[e.URL] = &e
		return nil
	})
}

func (c *Cache) writeDisk(e *Entry) error {
	path := filepath.Join(c.dir, base64.RawURLEncoding.EncodeToString([]byte(e.URL))+".json")
	f, err := os.CreateTemp(c.dir, "tmp-*")
	if err != nil {
		return err
	}

	if err := json.NewEncoder(f).Encode(e); err != nil {
		return errors.Join(err, f.Close())
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// understoodStatusCodes are the response status codes whose caching requirements this cache understands,
// as required by the must-understand directive (RFC 9111, Section 5.2.2.2). We only cache successful responses.
var understoodStatusCodes = map[int]struct{}{
	http.StatusOK: {},
}

func freshness(status int, header http.Header, body []byte) time.Duration {
	if d, ok := crlFreshness(body); ok {
		return d
	}
	if d, ok := cacheControlMaxAge(status, header); ok {
		return d
	}
	return defaultTTL
}

func cacheControlMaxAge(status int, header http.Header) (time.Duration, bool) {
	cc := header.Get("Cache-Control")
	if cc == "" {
		return 0, false
	}
	directives := parseCacheControl(cc)

	_, mustUnderstand := directives["must-understand"]
	_, understood := understoodStatusCodes[status]
	honorMustUnderstand := mustUnderstand && understood

	if _, ok := directives["no-cache"]; ok {
		return 0, false
	}
	if _, ok := directives["no-store"]; ok && !honorMustUnderstand {
		return 0, false
	}

	maxAge, ok := directives["max-age"]
	if !ok {
		return 0, false
	}
	secs, err := strconv.Atoi(maxAge)
	if err != nil || secs <= 0 {
		return 0, false
	}
	return time.Duration(secs) * time.Second, true
}

func parseCacheControl(cc string) map[string]string {
	directives := map[string]string{}
	for part := range strings.SplitSeq(cc, ",") {
		name, value, _ := strings.Cut(strings.TrimSpace(part), "=")
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		directives[name] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return directives
}

func crlFreshness(body []byte) (time.Duration, bool) {
	if len(body) == 0 {
		return 0, false
	}
	der := body
	if block, _ := pem.Decode(body); block != nil {
		der = block.Bytes
	}
	crl, err := x509.ParseRevocationList(der)
	if err != nil || crl.NextUpdate.IsZero() {
		return 0, false
	}
	d := time.Until(crl.NextUpdate)
	if d < 0 {
		return 0, false
	}
	return d, true
}
