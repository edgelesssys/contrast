// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/sdk/apiv1"
)

// capabilitiesPath is the path of the Coordinator's capabilities endpoint.
//
// This endpoint is deliberately unversioned. It's how clients discover which versions
// exist, so it must be reachable without knowing a version first.
const capabilitiesPath = "/capabilities"

// supportedAPIVersions are the API versions this SDK can speak, newest first.
var supportedAPIVersions = []string{apiv1.Version}

// negotiation caches the result of API version negotiation.
//
// It's held behind a pointer so that copying a [Client] doesn't copy the mutex.
type negotiation struct {
	mu      sync.Mutex
	version string
}

// NegotiateAPIVersion returns the newest API version supported by both this SDK and the Coordinator.
//
// The result is cached, so this costs at most one request per [Client].
func (c Client) NegotiateAPIVersion(ctx context.Context) (string, error) {
	c.negotiated.mu.Lock()
	defer c.negotiated.mu.Unlock()
	if c.negotiated.version != "" {
		return c.negotiated.version, nil
	}

	body, err := c.transport.DoJSON(ctx, http.MethodGet, capabilitiesPath, nil)
	if err != nil {
		return "", fmt.Errorf("getting capabilities: %w", err)
	}
	var caps apitypes.CapabilitiesResponse
	if err := json.Unmarshal(body, &caps); err != nil {
		return "", fmt.Errorf("unmarshalling capabilities: %w", err)
	}

	// supportedAPIVersions is ordered newest first, so the first match is the best one.
	for _, version := range supportedAPIVersions {
		if slices.Contains(caps.APIVersions, version) {
			c.negotiated.version = version
			return version, nil
		}
	}
	return "", fmt.Errorf("no common API version: Coordinator supports %v, SDK supports %v", caps.APIVersions, supportedAPIVersions)
}

// V1 returns a client for version v1 of the Coordinator's HTTP API.
func (c Client) V1() *apiv1.API {
	return apiv1.New(c.transport)
}
