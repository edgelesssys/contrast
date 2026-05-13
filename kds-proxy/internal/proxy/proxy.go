// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/edgelesssys/contrast/kds-proxy/internal/allowlist"
	"github.com/edgelesssys/contrast/kds-proxy/internal/ca"
	"github.com/edgelesssys/contrast/kds-proxy/internal/cache"
	"github.com/edgelesssys/contrast/kds-proxy/internal/upstream"
)

// AllowFunc decides whether the proxy will tunnel to host.
type AllowFunc func(host string) bool

// Server is the HTTP-level proxy server.
type Server struct {
	log      *slog.Logger
	ca       *ca.CA
	cache    *cache.Cache
	upstream *upstream.Fetcher
	allows   AllowFunc

	hits     atomic.Uint64
	misses   atomic.Uint64
	stale    atomic.Uint64
	rejected atomic.Uint64
	errors   atomic.Uint64
}

// New constructs a Server. If allows is nil, the default allowlist is used.
func New(log *slog.Logger, ca *ca.CA, c *cache.Cache, u *upstream.Fetcher, allows AllowFunc) *Server {
	if allows == nil {
		allows = allowlist.Allows
	}
	return &Server{log: log, ca: ca, cache: c, upstream: u, allows: allows}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodConnect:
		s.handleConnect(w, r)
	case http.MethodGet:
		switch r.URL.Path {
		case "/healthz":
			_, _ = w.Write([]byte("ok"))
		case "/metrics":
			s.writeMetrics(w)
		case "/ca.crt":
			w.Header().Set("Content-Type", "application/x-pem-file")
			_, _ = w.Write(s.ca.CertPEM())
		default:
			http.Error(w, "kds-proxy: direct requests not supported", http.StatusBadRequest)
		}
	default:
		s.rejected.Add(1)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Host
	if host == "" {
		host = r.Host
	}
	sniHost := stripPort(host)
	if !s.allows(sniHost) {
		s.rejected.Add(1)
		s.log.Warn("rejecting CONNECT to disallowed host", "host", host)
		http.Error(w, "host not allowed", http.StatusForbidden)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		s.log.Error("hijack failed", "err", err)
		return
	}
	defer clientConn.Close()

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		return
	}

	certPEM, keyPEM, err := s.ca.LeafPEM(sniHost)
	if err != nil {
		s.log.Error("minting leaf", "host", sniHost, "err", err)
		return
	}
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		s.log.Error("loading leaf keypair", "err", err)
		return
	}
	tlsConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	})
	if err := tlsConn.HandshakeContext(r.Context()); err != nil {
		s.log.Debug("TLS handshake with client failed", "host", sniHost, "err", err)
		return
	}
	defer tlsConn.Close()

	s.serveTunneled(r.Context(), tlsConn, sniHost)
}

func (s *Server) serveTunneled(ctx context.Context, conn net.Conn, host string) {
	br := bufio.NewReader(conn)
	for {
		req, err := http.ReadRequest(br)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.log.Debug("reading tunneled request", "host", host, "err", err)
			}
			return
		}
		if req.Method != http.MethodGet {
			s.rejected.Add(1)
			writeStatus(conn, http.StatusMethodNotAllowed, "only GET allowed")
			return
		}
		req.URL.Scheme = "https"
		req.URL.Host = host
		fullURL := req.URL.String()

		if entry, fresh := s.cache.Get(fullURL); fresh {
			s.hits.Add(1)
			s.log.Info("cache hit", "url", fullURL)
			writeRaw(conn, entry.Status, entry.Header, entry.Body)
			continue
		}

		res, err := s.upstream.Get(ctx, fullURL)
		if err != nil {
			if entry, _ := s.cache.Get(fullURL); entry != nil {
				s.stale.Add(1)
				s.log.Warn("serving stale on upstream error", "url", fullURL, "err", err)
				writeRaw(conn, entry.Status, entry.Header, entry.Body)
				continue
			}
			s.errors.Add(1)
			s.log.Error("upstream fetch failed and no cache entry", "url", fullURL, "err", err)
			writeStatus(conn, http.StatusBadGateway, "upstream unavailable")
			return
		}

		s.misses.Add(1)
		s.log.Info("cache miss, fetched upstream", "url", fullURL, "status", res.Status)
		entry, err := s.cache.Put(fullURL, res.Status, res.Header, res.Body)
		if err != nil {
			s.log.Error("cache write failed", "url", fullURL, "err", err)
			writeRaw(conn, res.Status, res.Header, res.Body)
			continue
		}
		writeRaw(conn, entry.Status, entry.Header, entry.Body)
	}
}

func (s *Server) writeMetrics(w http.ResponseWriter) {
	fmt.Fprintf(w, "kds_proxy_cache_hits %d\n", s.hits.Load())
	fmt.Fprintf(w, "kds_proxy_cache_misses %d\n", s.misses.Load())
	fmt.Fprintf(w, "kds_proxy_cache_stale %d\n", s.stale.Load())
	fmt.Fprintf(w, "kds_proxy_rejected %d\n", s.rejected.Load())
	fmt.Fprintf(w, "kds_proxy_upstream_errors %d\n", s.errors.Load())
}

func writeRaw(conn net.Conn, status int, header http.Header, body []byte) {
	resp := &http.Response{
		Status:        http.StatusText(status),
		StatusCode:    status,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
	_ = resp.Write(conn)
}

func writeStatus(conn net.Conn, status int, msg string) {
	h := http.Header{}
	h.Set("Content-Type", "text/plain; charset=utf-8")
	writeRaw(conn, status, h, []byte(msg))
}

func stripPort(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}
