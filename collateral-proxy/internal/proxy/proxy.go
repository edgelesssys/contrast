// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package proxy

import (
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/edgelesssys/contrast/collateral-proxy/internal/cache"
	"github.com/edgelesssys/contrast/collateral-proxy/internal/upstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metrics holds the Prometheus collectors for a Server, labeled by document and result.
type metrics struct {
	// requests counts proxied requests by result (hit, miss, stale, rejected, error).
	requests *prometheus.CounterVec
	// upstream counts upstream fetch outcomes by HTTP status code, or "error" when the fetch itself failed.
	upstream *prometheus.CounterVec
}

var (
	documentTypes  = []string{"crl", "ak-cert", "collateral", "unknown"}
	requestResults = []string{"hit", "miss", "stale", "error"}
)

func newMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		requests: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "collateral_proxy",
			Name:      "requests_total",
			Help:      "Proxied requests by result and document type.",
		}, []string{"result", "document"}),
		upstream: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "collateral_proxy",
			Name:      "upstream_responses_total",
			Help:      `Upstream fetch outcomes by HTTP status code (or "error") and document type.`,
		}, []string{"code", "document"}),
	}
	for _, t := range documentTypes {
		for _, r := range requestResults {
			m.requests.WithLabelValues(r, t)
		}
		m.upstream.WithLabelValues("error", t)
	}
	m.requests.WithLabelValues("rejected", "unknown")
	return m
}

// Server is a read-through caching forward proxy.
type Server struct {
	log         *slog.Logger
	cache       *cache.Cache
	upstream    *upstream.Fetcher
	metrics     *metrics
	metricsHTTP http.Handler
}

// New constructs a Server. If reg is nil, a fresh registry is created.
func New(log *slog.Logger, c *cache.Cache, u *upstream.Fetcher, reg *prometheus.Registry) *Server {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	return &Server{
		log:         log,
		cache:       c,
		upstream:    u,
		metrics:     newMetrics(reg),
		metricsHTTP: promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
	}
}

// route resolves a request path to the vendor host that serves it  and a coarse document type used as a metrics label.
func route(path string) (host, docType string, ok bool) {
	switch {
	case strings.HasPrefix(path, "/vcek/"), strings.HasPrefix(path, "/vlek/"):
		return "kdsintf.amd.com", amdKDSDocType(path), true
	case strings.HasPrefix(path, "/sgx/"), strings.HasPrefix(path, "/tdx/"):
		return "api.trustedservices.intel.com", intelPCSDocType(path), true
	case strings.HasPrefix(path, "/IntelSGX"):
		return "certificates.trustedservices.intel.com", intelCertsDocType(path), true
	case strings.HasPrefix(path, "/v1/rim/"):
		return "rim.attestation.nvidia.com", "collateral", true
	default:
		return "", "", false
	}
}

// amdHardwareID matches the hex hardware ID that addresses a VCEK/VLEK certificate.
var amdHardwareID = regexp.MustCompile(`^[0-9a-fA-F]+$`)

// amdKDSDocType classifies AMD KDS paths of the form /{vcek,vlek}/v1/{product}/{resource}.
func amdKDSDocType(path string) string {
	switch seg := lastSegment(path); {
	case seg == "crl":
		return "crl"
	case seg == "cert_chain":
		return "collateral"
	case amdHardwareID.MatchString(seg):
		return "ak-cert"
	default:
		return "unknown"
	}
}

// intelPCSDocType classifies Intel PCS paths by their trailing resource.
func intelPCSDocType(path string) string {
	switch seg := lastSegment(path); seg {
	case "pckcrl", "rootcacrl":
		return "crl"
	case "pckcert":
		return "ak-cert"
	case "pckcerts", "tcb", "identity": // identity covers qe/qve/tdqe
		return "collateral"
	default:
		return "unknown"
	}
}

// intelCertsDocType classifies the Intel SGX Root CA distribution host.
func intelCertsDocType(path string) string {
	if lastSegment(path) == "IntelSGXRootCA.der" {
		return "crl"
	}
	return "unknown"
}

func lastSegment(path string) string {
	return path[strings.LastIndex(path, "/")+1:]
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.metrics.requests.WithLabelValues("rejected", "unknown").Inc()
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	switch r.URL.Path {
	case "/healthz":
		_, _ = w.Write([]byte("ok"))
		return
	case "/metrics":
		s.metricsHTTP.ServeHTTP(w, r)
		return
	}
	upstreamHost, docType, ok := route(r.URL.Path)
	if !ok {
		s.metrics.requests.WithLabelValues("rejected", "unknown").Inc()
		s.log.Warn("rejecting request for unknown collateral path", "path", r.URL.Path)
		http.Error(w, "unknown collateral path", http.StatusNotFound)
		return
	}
	s.serveCollateral(w, r, upstreamHost, docType)
}

func (s *Server) serveCollateral(w http.ResponseWriter, r *http.Request, upstreamHost, docType string) {
	upstreamURL := "https://" + upstreamHost + r.URL.Path
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	entry, fresh := s.cache.Get(upstreamURL)
	if fresh {
		s.metrics.requests.WithLabelValues("hit", docType).Inc()
		s.log.Debug("cache hit", "url", upstreamURL)
		writeResponse(w, entry.Status, entry.Header, entry.Body)
		return
	}

	res, err := s.upstream.Get(r.Context(), upstreamURL)
	if err != nil {
		s.metrics.upstream.WithLabelValues("error", docType).Inc()
		if entry != nil {
			s.metrics.requests.WithLabelValues("stale", docType).Inc()
			s.log.Warn("serving stale on upstream error", "url", upstreamURL, "err", err)
			writeResponse(w, entry.Status, entry.Header, entry.Body)
			return
		}
		s.metrics.requests.WithLabelValues("error", docType).Inc()
		s.log.Error("upstream fetch failed and no cache entry", "url", upstreamURL, "err", err)
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}

	s.metrics.requests.WithLabelValues("miss", docType).Inc()
	s.metrics.upstream.WithLabelValues(strconv.Itoa(res.Status), docType).Inc()
	s.log.Info("cache miss, fetched upstream", "url", upstreamURL, "status", res.Status)
	entry, err = s.cache.Put(upstreamURL, res.Status, res.Header, res.Body)
	if err != nil {
		s.log.Error("cache write failed", "url", upstreamURL, "err", err)
		writeResponse(w, res.Status, res.Header, res.Body)
		return
	}
	writeResponse(w, entry.Status, entry.Header, entry.Body)
}

// hopByHopHeaders are not forwarded from the upstream response to the client.
var hopByHopHeaders = map[string]struct{}{
	"Connection":        {},
	"Transfer-Encoding": {},
	"Content-Length":    {}, // set by the ResponseWriter when the body is written
}

func writeResponse(w http.ResponseWriter, status int, header http.Header, body []byte) {
	for k, vs := range header {
		if _, skip := hopByHopHeaders[http.CanonicalHeaderKey(k)]; skip {
			continue
		}
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
