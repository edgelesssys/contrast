// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/edgelesssys/contrast/internal/atls/validators"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/sdk/internal/httpapi"
	"github.com/spf13/afero"
)

// Client is used to interact with a Contrast deployment.
type Client struct {
	// httpapi performs the HTTP requests against the Coordinator's API.
	httpapi *httpapi.Client

	// fsstore is the underlying filesystem-backed cache used by the Client.
	fsstore *fsstore.Store

	// collateralProxy, when non-empty, is the base URL of a proxy that attestation-collateral fetches are routed through.
	collateralProxy string

	log *slog.Logger

	// negotiateMu guards negotiatedVersion and capabilitiesDigest.
	negotiateMu sync.Mutex
	// negotiatedVersion caches the API version agreed on with the Coordinator,
	// empty until negotiated or pinned via [Client.WithAPIVersion].
	negotiatedVersion string
	// capabilitiesDigest is the SHA-256 digest of the raw capabilities response body received
	// from the Coordinator, nil until one was fetched.
	capabilitiesDigest []byte

	// validatorsFromManifestOverride is used by tests to replace the validators.
	validatorsFromManifestOverride func(*certcache.CachedHTTPSGetter, *manifest.Manifest, *slog.Logger) (validators.Validator, error)
}

// New returns a new [Client].
//
// baseURL is the root of the Coordinator's HTTP API, e.g. "http://coordinator:1314".
//
// Logging is disabled by default, and a memory-backed cache is used.
// For HTTP interactions, [http.DefaultClient] is used by default.
func New(baseURL string) *Client {
	c := &Client{
		log: slog.New(slog.DiscardHandler),
	}
	c.httpapi = &httpapi.Client{
		HTTPClient: http.DefaultClient,
		BaseURL:    baseURL,
		Log:        c.log,
	}
	c.fsstore = fsstore.New(afero.NewMemMapFs(), c.log.WithGroup("cert-cache"))
	return c
}

// WithAPIVersion pins the API version used by the Client's version-independent methods,
// skipping negotiation with the Coordinator.
//
// Calls fail if the Coordinator doesn't support the pinned version.
func (c *Client) WithAPIVersion(version string) *Client {
	c.negotiatedVersion = version
	return c
}

// WithFSStore replaces the Client's default filesystem-backed cache
// with one at the root of the given [afero.Fs].
//
// The store is instantiated at the root of `fs`, so [afero.newOsFs]
// should not be used directly. Instead, use [afero.NewBasePathFs].
func (c *Client) WithFSStore(fs afero.Fs) *Client {
	// TODO(burgerdev): This logger may be overridden via WithSlog,
	// depending on the call order.
	c.fsstore = fsstore.New(fs, c.log.WithGroup("cert-cache"))
	return c
}

// WithSlog replaces the Client's default [slog.Logger].
//
// The logger must not be nil.
func (c *Client) WithSlog(log *slog.Logger) *Client {
	c.log = log
	c.httpapi.Log = log
	return c
}

// WithHTTPClient replaces the Client's default [http.Client].
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	c.httpapi.HTTPClient = httpClient
	return c
}

// WithCollateralProxy routes the Client's attestation-collateral fetches (AMD KDS, Intel PCS, NVIDIA RIM)
// through a caching proxy at the given base URL, falling back to direct upstream fetching when the proxy is unreachable.
// An empty URL (the default) fetches directly upstream.
func (c *Client) WithCollateralProxy(proxyURL string) *Client {
	c.collateralProxy = proxyURL
	return c
}
