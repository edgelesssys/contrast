// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cache

import (
	"net/http"
	"testing"
	"time"
)

func TestCacheControlMaxAge(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   time.Duration
		ok     bool
	}{
		{"empty", "", 0, false},
		{"max-age 60", "max-age=60", 60 * time.Second, true},
		{"public max-age", "public, max-age=3600", 3600 * time.Second, true},
		{"no-store", "no-store, max-age=60", 0, false},
		{"no-cache", "no-cache", 0, false},
		{"zero", "max-age=0", 0, false},
		{"garbage", "max-age=NaN", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := http.Header{}
			if c.header != "" {
				h.Set("Cache-Control", c.header)
			}
			got, ok := cacheControlMaxAge(h)
			if ok != c.ok {
				t.Fatalf("ok=%v want %v", ok, c.ok)
			}
			if got != c.want {
				t.Fatalf("dur=%v want %v", got, c.want)
			}
		})
	}
}

func TestPutGetRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set("Cache-Control", "max-age=120")
	if _, err := c.Put("https://example/x", 200, h, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	e, fresh := c.Get("https://example/x")
	if e == nil || !fresh {
		t.Fatalf("expected fresh entry, got %+v fresh=%v", e, fresh)
	}
	if string(e.Body) != "hello" {
		t.Fatalf("body=%q", e.Body)
	}

	// Reopen from disk.
	c2, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	e2, fresh2 := c2.Get("https://example/x")
	if e2 == nil || !fresh2 || string(e2.Body) != "hello" {
		t.Fatalf("disk reload failed: %+v fresh=%v", e2, fresh2)
	}
}

func TestStaleEntryStillReturned(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set("Cache-Control", "max-age=1")
	e, err := c.Put("https://example/x", 200, h, []byte("hi"))
	if err != nil {
		t.Fatal(err)
	}
	e.FreshUntil = time.Now().Add(-time.Hour)
	got, fresh := c.Get("https://example/x")
	if got == nil {
		t.Fatal("entry vanished")
	}
	if fresh {
		t.Fatal("expected stale, got fresh")
	}
}
