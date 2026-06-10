// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/kds-proxy/internal/ca"
	"github.com/edgelesssys/contrast/kds-proxy/internal/cache"
	"github.com/edgelesssys/contrast/kds-proxy/internal/upstream"
)

func TestForwardProxyEndToEnd(t *testing.T) {
	var upstreamHits atomic.Int64
	upstreamSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits.Add(1)
		w.Header().Set("Cache-Control", "max-age=3600")
		_, _ = fmt.Fprintf(w, "vcek bytes for %s", r.URL.Path)
	}))
	defer upstreamSrv.Close()
	upstreamURL, err := url.Parse(upstreamSrv.URL)
	if err != nil {
		t.Fatal(err)
	}

	dialer := &net.Dialer{}
	fetchClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if strings.HasPrefix(addr, "kdsintf.amd.com:") {
					return dialer.DialContext(ctx, network, upstreamURL.Host)
				}
				return dialer.DialContext(ctx, network, addr)
			},
		},
		Timeout: 5 * time.Second,
	}

	authority, err := ca.LoadOrGenerate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cch, err := cache.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := New(slog.New(slog.DiscardHandler),
		authority, cch, upstream.New(fetchClient), nil)

	proxySrv := httptest.NewServer(srv)
	defer proxySrv.Close()
	proxyURL, _ := url.Parse(proxySrv.URL)

	clientPool := x509.NewCertPool()
	if ok := clientPool.AppendCertsFromPEM(authority.CertPEM()); !ok {
		t.Fatal("failed to append CA to pool")
	}
	clientCC := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{RootCAs: clientPool},
		},
		Timeout: 5 * time.Second,
	}

	body1 := mustGet(t, clientCC, "https://kdsintf.amd.com/vcek/v1/Milan/abc")
	if want := "vcek bytes for /vcek/v1/Milan/abc"; body1 != want {
		t.Fatalf("body=%q want %q", body1, want)
	}
	if got := upstreamHits.Load(); got != 1 {
		t.Fatalf("upstream hits=%d want 1", got)
	}

	body2 := mustGet(t, clientCC, "https://kdsintf.amd.com/vcek/v1/Milan/abc")
	if body2 != body1 {
		t.Fatalf("body mismatch")
	}
	if got := upstreamHits.Load(); got != 1 {
		t.Fatalf("upstream hits=%d want still 1", got)
	}
}

func TestRejectsDisallowedHost(t *testing.T) {
	authority, err := ca.LoadOrGenerate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cch, err := cache.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := New(slog.New(slog.DiscardHandler),
		authority, cch, upstream.New(&http.Client{Timeout: time.Second}), nil)
	proxySrv := httptest.NewServer(srv)
	defer proxySrv.Close()
	proxyURL, _ := url.Parse(proxySrv.URL)

	c := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   2 * time.Second,
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://evil.example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected error connecting to disallowed host")
	}
}

func mustGet(t *testing.T, c *http.Client, url string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build request %s: %v", url, err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	return string(b)
}
