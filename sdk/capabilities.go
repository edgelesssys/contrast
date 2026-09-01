// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/sdk/apiv1"
)

// capabilitiesPath is the path of the Coordinator's capabilities endpoint.
//
// This endpoint is deliberately unversioned. It's how clients discover which versions
// exist, so it must be reachable without knowing a version first.
const capabilitiesPath = "/capabilities"

// supportedAPIVersions are the API versions this SDK can speak, newest first.
var supportedAPIVersions = []string{apiv1.Version}

// NegotiateAPIVersion returns the newest API version supported by both this SDK and the Coordinator.
//
// If the expected manifest pins a MinimumAPIVersion, negotiation fails rather than settle on an older version.
//
// The first successful result is cached, so this costs at most one successful request per [Client].
func (c *Client) NegotiateAPIVersion(ctx context.Context) (string, error) {
	c.negotiateMu.Lock()
	defer c.negotiateMu.Unlock()
	if c.negotiatedVersion != "" {
		if err := enforceMinimumAPIVersion(c.negotiatedVersion, c.expectedManifest); err != nil {
			return "", err
		}
		return c.negotiatedVersion, nil
	}

	caps, err := c.getCapabilitiesLocked(ctx)
	if err != nil {
		return "", err
	}

	for _, version := range supportedAPIVersions {
		if !slices.Contains(caps.APIVersions, version) {
			continue
		}
		if err := enforceMinimumAPIVersion(version, c.expectedManifest); err != nil {
			return "", fmt.Errorf("refusing to negotiate: %w", err)
		}
		c.negotiatedVersion = version
		return version, nil
	}
	return "", fmt.Errorf("no common API version: Coordinator supports %v, SDK supports %v", caps.APIVersions, supportedAPIVersions)
}

// getCapabilitiesLocked fetches and parses the Coordinator's capabilities.
func (c *Client) getCapabilitiesLocked(ctx context.Context) (*apitypes.CapabilitiesResponse, error) {
	body, err := c.httpapi.DoJSON(ctx, http.MethodGet, capabilitiesPath, nil)
	if err != nil {
		return nil, fmt.Errorf("getting capabilities: %w", err)
	}
	var caps apitypes.CapabilitiesResponse
	if err := json.Unmarshal(body, &caps); err != nil {
		return nil, fmt.Errorf("unmarshalling capabilities: %w", err)
	}
	digest := sha256.Sum256(body)
	c.capabilitiesDigest = digest[:]
	return &caps, nil
}

// getCapabilitiesDigest returns the SHA-256 digest of the raw capabilities response body.
func (c *Client) getCapabilitiesDigest(ctx context.Context) ([]byte, error) {
	c.negotiateMu.Lock()
	defer c.negotiateMu.Unlock()
	if c.capabilitiesDigest == nil {
		if _, err := c.getCapabilitiesLocked(ctx); err != nil {
			return nil, err
		}
	}
	return c.capabilitiesDigest, nil
}

// enforceMinimumAPIVersion returns an error if version is older than the given manifest's optional MinimumAPIVersion pin.
func enforceMinimumAPIVersion(version string, m *manifest.Manifest) error {
	if m == nil || m.MinimumAPIVersion == "" {
		return nil
	}
	minVersion, err := apitypes.ParseAPIVersion(m.MinimumAPIVersion)
	if err != nil {
		return fmt.Errorf("parsing the manifest's MinimumAPIVersion: %w", err)
	}
	v, err := apitypes.ParseAPIVersion(version)
	if err != nil {
		return fmt.Errorf("parsing API version %q: %w", version, err)
	}
	if v < minVersion {
		return fmt.Errorf("API version %s is older than the minimum %s required by the manifest", version, m.MinimumAPIVersion)
	}
	return nil
}

// V1 returns a client for version v1 of the Coordinator's HTTP API.
func (c *Client) V1() *apiv1.API {
	return apiv1.New(c.httpapi)
}
