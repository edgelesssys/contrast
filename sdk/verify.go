// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/cryptohelpers"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/httpapi"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/spf13/afero"
)

// Client is used to interact with a Contrast deployment.
type Client struct {
	// httpClient will be used to contact the Coordinator HTTP API.
	// If nil, http.DefaultClient will be used.
	httpClient *http.Client

	// fsstore is the underlying filesystem-backed cache used by the
	// Client.
	fsstore *fsstore.Store

	log *slog.Logger

	// validatorsFromManifestOverride is used by tests to replace the validators.
	validatorsFromManifestOverride func(*certcache.CachedHTTPSGetter, *manifest.Manifest, *slog.Logger) ([]atls.Validator, error)
}

// New returns a new [Client].
//
// Logging is disabled by default, and a memory-backed cache is used.
// For HTTP interactions, [http.DefaultClient] is used by default.
func New() *Client {
	c := &Client{
		log:        slog.New(slog.DiscardHandler),
		httpClient: http.DefaultClient,
	}
	c.fsstore = fsstore.New(afero.NewMemMapFs(), c.log.WithGroup("cert-cache"))
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
	return c
}

// WithHTTPClient replaces the Client's default [http.Client].
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	c.httpClient = httpClient
	return c
}

// GetAttestation requests attestation evidence from the Coordinator's HTTP API.
//
// The URL needs to map to the http://coordinator:1314/attest endpoint, but can be reverse-proxied
// or HTTPS-enabled.
//
// The nonce needs to be exactly 32 bytes, which should come from a CSPRNG.
func (c Client) GetAttestation(ctx context.Context, url string, nonce []byte) ([]byte, error) {
	if len(nonce) != cryptohelpers.RNGLengthDefault {
		return nil, fmt.Errorf("bad nonce length: got %d, want %d", len(nonce), cryptohelpers.RNGLengthDefault)
	}
	body, err := json.Marshal(&httpapi.AttestationRequest{Nonce: nonce})
	if err != nil {
		return nil, fmt.Errorf("creating request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, fmt.Errorf("constructing HTTP request: %w", err)
	}

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		errBody, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response (status code %d): %w", httpResp.StatusCode, err)
		}
		details := httpResp.Status
		var resp httpapi.AttestationError
		if err := json.Unmarshal(errBody, &resp); err == nil {
			details = resp.Err
		} else {
			c.log.Error("parsing error response", "err", err, "response", string(errBody))
		}
		return nil, fmt.Errorf("HTTP API call failed with %d (%s): %s", httpResp.StatusCode, http.StatusText(httpResp.StatusCode), details)
	}
	resp, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading HTTP response body: %w", err)
	}
	return resp, nil
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

	resp, err := httpapi.UnmarshalAttestationResponse(attestation)
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

	kdsGetter := certcache.NewCachedHTTPSGetter(c.fsstore, certcache.NeverGCTicker, c.log.WithGroup("kds-getter"))
	validatorsFromManifest := ValidatorsFromManifest
	if c.validatorsFromManifestOverride != nil {
		validatorsFromManifest = c.validatorsFromManifestOverride
	}
	validators, err := validatorsFromManifest(kdsGetter, &latestManifest, c.log)
	if err != nil {
		return nil, fmt.Errorf("getting validators: %w", err)
	}

	transitions := buildTransitionChain(resp.Manifests)
	transitionDigest := transitions[len(transitions)-1].Digest()
	reportData := httpapi.ConstructReportData(nonce, transitionDigest[:], &resp.CoordinatorState)

	validated := false
	var errs []error
	for _, v := range validators {
		if err := v.Validate(ctx, resp.RawAttestationDoc, reportData[:]); err != nil {
			c.log.Debug("validator failed", "error", err)
			errs = append(errs, err)
			continue
		}
		validated = true
		break
	}
	if !validated {
		return nil, fmt.Errorf("validation failed:\n%w", errors.Join(errs...))
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

func buildTransitionChain(manifests [][]byte) []*history.Transition {
	transitions := make([]*history.Transition, 0, len(manifests))
	lastTransitionHash := [history.HashSize]byte{}
	for _, m := range manifests {
		md := history.Digest(m)
		t := &history.Transition{
			PreviousTransitionHash: lastTransitionHash,
			ManifestHash:           md,
		}
		transitions = append(transitions, t)
		lastTransitionHash = t.Digest()
	}
	return transitions
}
