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
	contrastcrypto "github.com/edgelesssys/contrast/internal/crypto"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/httpapi"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
)

// Client is used to interact with a Contrast deployment.
type Client struct {
	// HTTPClient will be used to contact the Coordinator HTTP API.
	// If nil, http.DefaultClient will be used.
	HTTPClient *http.Client

	log *slog.Logger
}

// New returns a Client with logging disabled.
func New() Client {
	return Client{
		log: slog.New(slog.DiscardHandler),
	}
}

// NewWithSlog can be used to configure how the SDK logs messages.
func NewWithSlog(log *slog.Logger) Client {
	return Client{
		log: log,
	}
}

// GetCoordinatorState calls GetManifests on the coordinator's userapi via aTLS.
func (c Client) GetCoordinatorState(ctx context.Context, kdsDir string, manifestBytes []byte, endpoint string) (httpapi.CoordinatorState, error) {
	var m manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return httpapi.CoordinatorState{}, fmt.Errorf("unmarshalling manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return httpapi.CoordinatorState{}, fmt.Errorf("validating manifest: %w", err)
	}

	kdsCache := fsstore.New(kdsDir, c.log.WithGroup("kds-cache"))
	kdsGetter := certcache.NewCachedHTTPSGetter(kdsCache, certcache.NeverGCTicker, c.log.WithGroup("kds-getter"))
	validators, err := ValidatorsFromManifest(kdsGetter, &m, c.log)
	if err != nil {
		return httpapi.CoordinatorState{}, fmt.Errorf("getting validators: %w", err)
	}
	dialer := dialer.New(atls.NoIssuer, validators, atls.NoMetrics, nil, c.log)

	c.log.Debug("Dialing coordinator", "endpoint", endpoint)

	conn, err := dialer.Dial(ctx, endpoint)
	if err != nil {
		return httpapi.CoordinatorState{}, fmt.Errorf("dialing coordinator: %w", err)
	}
	defer conn.Close()

	c.log.Debug("Getting manifest")

	client := userapi.NewUserAPIClient(conn)
	resp, err := client.GetManifests(ctx, &userapi.GetManifestsRequest{})
	if err != nil {
		return httpapi.CoordinatorState{}, fmt.Errorf("getting manifests: %w", err)
	}

	return httpapi.CoordinatorState{
		Manifests: resp.Manifests,
		Policies:  resp.Policies,
		RootCA:    resp.RootCA,
		MeshCA:    resp.MeshCA,
	}, nil
}

// Verify checks if a given manifest is the latest manifest in the given history.
// The expected manifest should be supplied by the caller, the history should be received from the coordinator.
func (Client) Verify(expectedManifest []byte, manifestHistory [][]byte) error {
	if len(manifestHistory) == 0 {
		return fmt.Errorf("manifest history is empty")
	}

	currentManifest := manifestHistory[len(manifestHistory)-1]
	if !bytes.Equal(currentManifest, expectedManifest) {
		return fmt.Errorf("active manifest does not match expected manifest")
	}

	return nil
}

// GetAttestation requests attestation evidence from the Coordinator's HTTP API.
//
// The URL needs to map to the http://coordinator:1314/attest endpoint, but can be reverse-proxied
// or HTTPS-enabled.
//
// The nonce needs to be exactly 32 bytes, which should come from a CSPRNG.
func (c Client) GetAttestation(ctx context.Context, url string, nonce []byte) ([]byte, error) {
	if len(nonce) != contrastcrypto.RNGLengthDefault {
		return nil, fmt.Errorf("bad nonce length: got %d, want %d", len(nonce), contrastcrypto.RNGLengthDefault)
	}
	body, err := json.Marshal(&httpapi.AttestationRequest{Nonce: nonce})
	if err != nil {
		return nil, fmt.Errorf("creating request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("constructing HTTP request: %w", err)
	}
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(body)), nil }

	client := http.DefaultClient
	if c.HTTPClient != nil {
		client = c.HTTPClient
	}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		details := httpResp.Status
		var resp httpapi.AttestationError
		if err := json.NewDecoder(httpResp.Body).Decode(&resp); err == nil {
			details = resp.Err
		} else {
			c.log.Error("parsing error response", "err", err)
		}
		return nil, fmt.Errorf("HTTP API call failed with %d (%s): %s", httpResp.StatusCode, http.StatusText(httpResp.StatusCode), details)
	}
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, httpResp.Body); err != nil {
		return nil, fmt.Errorf("reading HTTP response body: %w", err)
	}
	return buf.Bytes(), nil
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
func (c Client) ValidateAttestation(ctx context.Context, kdsDir string, nonce []byte, attestation []byte) (*httpapi.CoordinatorState, error) {
	if len(nonce) != contrastcrypto.RNGLengthDefault {
		return nil, fmt.Errorf("wrong nonce length: got %d, want %d", len(nonce), contrastcrypto.RNGLengthDefault)
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

	kdsCache := fsstore.New(kdsDir, c.log.WithGroup("kds-cache"))
	kdsGetter := certcache.NewCachedHTTPSGetter(kdsCache, certcache.NeverGCTicker, c.log.WithGroup("kds-getter"))
	validators, err := ValidatorsFromManifest(kdsGetter, &latestManifest, c.log)
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
			errs = append(errs, err)
			continue
		}
		validated = true
		break
	}
	if !validated {
		return nil, fmt.Errorf("validation failed:\n%w", errors.Join(errs...))
	}
	return &resp.CoordinatorState, nil
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
