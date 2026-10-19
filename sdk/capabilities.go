// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/sdk/apiv1"
	"github.com/edgelesssys/contrast/sdk/internal/apiversion"
)

// SupportedAPIVersions are the API versions this SDK can speak, newest first.
//
// The Coordinator advertises this list on its capabilities endpoint.
var SupportedAPIVersions = []string{apiv1.Version}

// ErrNoCommonAPIVersion is returned if the SDK and the Coordinator share no API version.
var ErrNoCommonAPIVersion = errors.New("no common API version")

// ErrMinimumAPIVersionUnmet is returned if the API version in use is older than the pinned minimum API version.
var ErrMinimumAPIVersionUnmet = apiversion.ErrMinimumUnmet

// ErrAPIVersionDowngrade is returned if the capabilities the API version was negotiated with don't match the capabilities the Coordinator attested to.
var ErrAPIVersionDowngrade = apiversion.ErrDowngrade

// NegotiateAPIVersion returns the newest API version supported by both this SDK and the Coordinator.
//
// If the negotiated version is older than the one pinned via [Client.WithMinimumAPIVersion], negotiation fails with [ErrMinimumAPIVersionUnmet].
// If there is no shared version, it fails with [ErrNoCommonAPIVersion].
// If no version can be negotiated, only the unversioned legacy API is left, so negotiation also fails with
// [ErrMinimumAPIVersionUnmet] if a minimum API version is pinned.
//
// Use the returned version to pick the client of that version, like [Client.V1].
//
// The first successful result is cached, so this costs at most one successful request per [Client].
func (c *Client) NegotiateAPIVersion(ctx context.Context) (string, error) {
	c.negotiateMu.Lock()
	defer c.negotiateMu.Unlock()
	if c.negotiatedVersion != "" {
		if err := enforceMinimumAPIVersion(c.negotiatedVersion, c.minimumAPIVersion); err != nil {
			return "", err
		}
		return c.negotiatedVersion, nil
	}

	version, caps, err := c.negotiate(ctx)
	if err != nil {
		if pinErr := enforceMinimumAPIVersion(legacyAPIVersion, c.minimumAPIVersion); pinErr != nil {
			return "", fmt.Errorf("%w (negotiating API version: %w)", pinErr, err)
		}
		return "", err
	}
	if err := enforceMinimumAPIVersion(version, c.minimumAPIVersion); err != nil {
		return "", fmt.Errorf("refusing to negotiate: %w", err)
	}
	c.negotiatedVersion = version
	c.negotiatedCapabilities = caps
	return version, nil
}

// negotiate fetches the Coordinator's capabilities and returns the newest API version both sides support.
func (c *Client) negotiate(ctx context.Context) (string, *apitypes.CapabilitiesResponse, error) {
	body, err := c.httpapi.DoJSON(ctx, apitypes.CapabilitiesMethod, apitypes.CapabilitiesPath, nil)
	if err != nil {
		return "", nil, fmt.Errorf("getting capabilities: %w", err)
	}
	var caps apitypes.CapabilitiesResponse
	if err := json.Unmarshal(body, &caps); err != nil {
		return "", nil, fmt.Errorf("unmarshalling capabilities: %w", err)
	}
	version, ok := newestCommonAPIVersion(&caps)
	if !ok {
		return "", nil, fmt.Errorf("%w: Coordinator supports %v, SDK supports %v", ErrNoCommonAPIVersion, caps.APIVersions, SupportedAPIVersions)
	}
	return version, &caps, nil
}

// newestCommonAPIVersion returns the newest API version supported by both this SDK and
// a Coordinator with the given capabilities.
func newestCommonAPIVersion(caps *apitypes.CapabilitiesResponse) (string, bool) {
	// SupportedAPIVersions is ordered newest first, so the first match is the best one.
	for _, version := range SupportedAPIVersions {
		if slices.Contains(caps.APIVersions, version) {
			return version, true
		}
	}
	return "", false
}

const legacyAPIVersion = apiversion.Legacy

// enforceMinimumAPIVersion returns [ErrMinimumAPIVersionUnmet] if version is older than the optional minimum API version pin.
func enforceMinimumAPIVersion(version string, minimum string) error {
	return apiversion.EnforceMinimum(version, minimum)
}

// V1 returns a client for version v1 of the Coordinator's HTTP API.
//
// It shares the Client's configuration, including its minimum API version pin, see [Client.WithMinimumAPIVersion].
// If the Client negotiated the API version, see [Client.NegotiateAPIVersion], attestations validated by the
// returned client must attest to the capabilities that were negotiated with.
func (c *Client) V1() *apiv1.API {
	return apiv1.New(apiv1.Config{
		HTTP:                   c.httpapi,
		Validator:              c.coordinatorValidator,
		MinimumAPIVersion:      c.minimumAPIVersion,
		NegotiatedCapabilities: c.negotiatedCapabilitiesOrNil,
	})
}

// negotiatedCapabilitiesOrNil returns the capabilities the API version was negotiated with, or nil if it wasn't negotiated.
func (c *Client) negotiatedCapabilitiesOrNil() *apitypes.CapabilitiesResponse {
	c.negotiateMu.Lock()
	defer c.negotiateMu.Unlock()
	return c.negotiatedCapabilities
}
