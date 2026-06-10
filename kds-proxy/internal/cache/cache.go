// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cache

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
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

const minTTL = time.Hour

// Entry is a cached HTTP response.
type Entry struct {
	URL        string      `json:"url"`
	Status     int         `json:"status"`
	Header     http.Header `json:"header"`
	Body       []byte      `json:"body"`
	FreshUntil time.Time   `json:"freshUntil"`
}

// Cache is an in-memory + on-disk cache for upstream HTTP responses.
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
		FreshUntil: time.Now().Add(freshness(header, body)),
	}
	if err := c.writeDisk(e); err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.entries[url] = e
	c.mu.Unlock()
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
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var e Entry
		if err := json.Unmarshal(raw, &e); err != nil {
			slog.Warn("skipping corrupt cache entry", "path", path, "err", err)
			return nil
		}
		c.entries[e.URL] = &e
		return nil
	})
}

func (c *Cache) writeDisk(e *Entry) error {
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}
	path := filepath.Join(c.dir, keyHash(e.URL)+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func keyHash(url string) string {
	sum := sha256.Sum256([]byte(url))
	return hex.EncodeToString(sum[:])
}

func freshness(header http.Header, body []byte) time.Duration {
	if d, ok := crlFreshness(body); ok {
		return d
	}
	if d, ok := cacheControlMaxAge(header); ok {
		return d
	}
	return minTTL
}

func cacheControlMaxAge(header http.Header) (time.Duration, bool) {
	cc := header.Get("Cache-Control")
	if cc == "" {
		return 0, false
	}
	if strings.Contains(cc, "no-store") || strings.Contains(cc, "no-cache") {
		return 0, false
	}
	for _, part := range strings.Split(cc, ",") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "max-age=") {
			continue
		}
		secs, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
		if err != nil || secs <= 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	return 0, false
}

func crlFreshness(body []byte) (time.Duration, bool) {
	if len(body) == 0 {
		return 0, false
	}
	crl, err := x509.ParseRevocationList(body)
	if err != nil || crl.NextUpdate.IsZero() {
		return 0, false
	}
	d := time.Until(crl.NextUpdate)
	if d < 0 {
		return 0, false
	}
	return d, true
}
