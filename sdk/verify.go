// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/internal/atls/validators"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/cryptohelpers"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/sdk/internal/transport"
	"github.com/spf13/afero"
)

// attestPath is the path of the Coordinator's attestation endpoint, which is unversioned.
const attestPath = "/attest"

// Client is used to interact with a Contrast deployment.
type Client struct {
	// transport performs the HTTP requests against the Coordinator's API.
	transport *transport.Client

	// fsstore is the underlying filesystem-backed cache used by the
	// Client.
	fsstore *fsstore.Store

	// collateralProxy, when non-empty, is the base URL of a proxy that attestation-collateral fetches are routed through.
	collateralProxy string

	log *slog.Logger

	// negotiated caches the API version agreed on with the Coordinator.
	negotiated *negotiation

	// validatorsFromManifestOverride is used by tests to replace the validators.
	validatorsFromManifestOverride func(*certcache.CachedHTTPSGetter, *manifest.Manifest, *slog.Logger) (validators.Validator, error)
}

// New returns a new [Client].
//
// Logging is disabled by default, and a memory-backed cache is used.
// For HTTP interactions, [http.DefaultClient] is used by default.
//
// The Coordinator's address must be set via [Client.WithBaseURL] before any API call.
func New() *Client {
	c := &Client{
		log:        slog.New(slog.DiscardHandler),
		negotiated: &negotiation{},
	}
	c.transport = &transport.Client{
		HTTPClient: http.DefaultClient,
		Log:        c.log,
	}
	c.fsstore = fsstore.New(afero.NewMemMapFs(), c.log.WithGroup("cert-cache"))
	return c
}

// WithBaseURL sets the root of the Coordinator's HTTP API, e.g. "http://coordinator:1314".
//
// The API paths are resolved against it, so a reverse proxy can be addressed by passing its
// URL, including any path prefix it serves Contrast under.
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.transport.BaseURL = baseURL
	return c
}

// WithAPIVersion pins the API version used by the Client's version-independent methods,
// skipping negotiation with the Coordinator.
//
// Calls fail if the Coordinator doesn't support the pinned version.
func (c *Client) WithAPIVersion(version string) *Client {
	c.negotiated.version = version
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
	c.transport.Log = log
	return c
}

// WithHTTPClient replaces the Client's default [http.Client].
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	c.transport.HTTPClient = httpClient
	return c
}

// WithCollateralProxy routes the Client's attestation-collateral fetches (AMD KDS, Intel PCS, NVIDIA RIM)
// through a caching proxy at the given base URL, falling back to direct upstream fetching when the proxy is unreachable.
// An empty URL (the default) fetches directly upstream.
func (c *Client) WithCollateralProxy(proxyURL string) *Client {
	c.collateralProxy = proxyURL
	return c
}

// GetAttestation requests attestation evidence from the Coordinator's HTTP API.
//
// It's served at the /attest path relative to the Client's base URL. The endpoint is not
// versioned, so this method doesn't negotiate an API version.
//
// The nonce needs to be exactly 32 bytes, which should come from a CSPRNG.
func (c Client) GetAttestation(ctx context.Context, nonce []byte) ([]byte, error) {
	if len(nonce) != cryptohelpers.RNGLengthDefault {
		return nil, fmt.Errorf("bad nonce length: got %d, want %d", len(nonce), cryptohelpers.RNGLengthDefault)
	}
	return c.transport.DoJSON(ctx, http.MethodPost, attestPath, &apitypes.AttestationRequest{Nonce: nonce})
}

// ValidateAttestation validates the Coordinator state returned by the http://coordinator:1314/attest endpoint.
//
// The input for this function should be the nonce passed into GetAttestation and the byte slice
// returned by it.
//
// If this function returns nil, validation passed and the caller can rely on the state.MeshCA
// issuing certificates according to the last entry of state.Manifests.
//
// Note: this function does not verify manifest content! It's the callers responsibility to compare
// the latest manifest with an expected manifest, if that exists, or verify that all manifest
// fields match their expectations.
func (c Client) ValidateAttestation(ctx context.Context, nonce []byte, attestation []byte) (*CoordinatorState, error) {
	if len(nonce) != cryptohelpers.RNGLengthDefault {
		return nil, fmt.Errorf("wrong nonce length: got %d, want %d", len(nonce), cryptohelpers.RNGLengthDefault)
	}

	resp, err := apitypes.UnmarshalAttestationResponse(attestation)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling attestation document: %w", err)
	}

	if len(resp.Manifests) == 0 {
		return nil, fmt.Errorf("coordinator state does not include manifests")
	}
	var latestManifest manifest.Manifest
	if err := json.Unmarshal(resp.Manifests[len(resp.Manifests)-1], &latestManifest); err != nil {
		return nil, fmt.Errorf("unmarshalling latest manifest: %w", err)
	}
	if err := latestManifest.Validate(); err != nil {
		return nil, fmt.Errorf("validating latest manifest: %w", err)
	}

	kdsGetter := certcache.NewCachedHTTPSGetter(c.fsstore, certcache.NeverGCTicker, c.log.WithGroup("kds-getter"), c.collateralProxy)
	validatorsFromManifest := func(kdsGetter *certcache.CachedHTTPSGetter, m *manifest.Manifest, log *slog.Logger) (validators.Validator, error) {
		return m.CoordinatorValidator(log, kdsGetter)
	}
	if c.validatorsFromManifestOverride != nil {
		validatorsFromManifest = c.validatorsFromManifestOverride
	}
	validator, err := validatorsFromManifest(kdsGetter, &latestManifest, c.log)
	if err != nil {
		return nil, fmt.Errorf("getting validators: %w", err)
	}

	transitions := history.BuildTransitionChain(resp.Manifests)
	transitionDigest := transitions[len(transitions)-1].Digest()
	reportData := apitypes.ConstructReportData(nonce, transitionDigest[:], &resp.CoordinatorState)

	if err := validator.Validate(ctx, resp.AttestationType, resp.RawAttestationDoc, reportData[:]); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	state := CoordinatorState{
		Manifests: resp.Manifests,
		Policies:  resp.Policies,
		RootCA:    resp.RootCA,
		MeshCA:    resp.MeshCA,
	}
	return &state, nil
}

// CoordinatorState represents the state of the Contrast Coordinator at a fixed point in time.
type CoordinatorState struct {
	// Manifests is a slice of manifests. It represents the manifest history of the Coordinator it was received from, sorted from oldest to newest.
	Manifests [][]byte
	// Policies contains all policies that have been referenced in any manifest in Manifests. Used to verify the guarantees a deployment had over its lifetime.
	Policies [][]byte
	// PEM-encoded certificate of the deployment's root CA.
	RootCA []byte
	// PEM-encoded certificate of the deployment's mesh CA.
	MeshCA []byte
	// Hash of the latest transition in the Coordinator's history.
	LatestTransitionHash []byte
	// Signature of the latest transition hash by the Coordinator.
	LatestTransitionSignature []byte
}
