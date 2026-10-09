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
)

// SupportedAPIVersions are the API versions this SDK can speak, newest first.
//
// The Coordinator advertises this list on its capabilities endpoint.
// TODO(charludo): empty because the Coordinator serves no versioned endpoint yet.
var SupportedAPIVersions = []string{}

// ErrNoCommonAPIVersion is returned if the SDK and the Coordinator share no API version.
var ErrNoCommonAPIVersion = errors.New("no common API version")

// ErrMinimumAPIVersionUnmet is returned if the API version in use is older than the pinned minimum API version.
var ErrMinimumAPIVersionUnmet = errors.New("minimum API version not met")

// ErrAPIVersionDowngrade is returned if the API version negotiation to doesn't match the capabilities the Coordinator attested to.
var ErrAPIVersionDowngrade = errors.New("API version was downgraded")

// NegotiateAPIVersion returns the newest API version supported by both this SDK and the Coordinator.
//
// If the negotiated version is older than the one pinned via [Client.WithMinimumAPIVersion], negotiation fails with [ErrMinimumAPIVersionUnmet].
// If there is no shared version, it fails with [ErrNoCommonAPIVersion].
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

	body, err := c.httpapi.DoJSON(ctx, apitypes.CapabilitiesMethod, apitypes.CapabilitiesPath, nil)
	if err != nil {
		return "", fmt.Errorf("getting capabilities: %w", err)
	}
	var caps apitypes.CapabilitiesResponse
	if err := json.Unmarshal(body, &caps); err != nil {
		return "", fmt.Errorf("unmarshalling capabilities: %w", err)
	}

	version, ok := newestCommonAPIVersion(&caps)
	if !ok {
		return "", fmt.Errorf("%w: Coordinator supports %v, SDK supports %v", ErrNoCommonAPIVersion, caps.APIVersions, SupportedAPIVersions)
	}
	if err := enforceMinimumAPIVersion(version, c.minimumAPIVersion); err != nil {
		return "", fmt.Errorf("refusing to negotiate: %w", err)
	}
	c.negotiatedVersion = version
	c.negotiatedCapabilities = &caps
	return version, nil
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

const legacyAPIVersion = ""

// enforceMinimumAPIVersion returns [ErrMinimumAPIVersionUnmet] if version is older than the optional minimum API version pin.
func enforceMinimumAPIVersion(version string, minimum string) error {
	if minimum == "" {
		return nil
	}
	minVersion, err := apitypes.ParseAPIVersion(minimum)
	if err != nil {
		return fmt.Errorf("parsing the minimum API version: %w", err)
	}
	if version == legacyAPIVersion {
		return fmt.Errorf("%w: the unversioned legacy API is older than the minimum %s required", ErrMinimumAPIVersionUnmet, minimum)
	}
	v, err := apitypes.ParseAPIVersion(version)
	if err != nil {
		return fmt.Errorf("parsing API version %q: %w", version, err)
	}
	if v < minVersion {
		return fmt.Errorf("%w: API version %s is older than the minimum %s required", ErrMinimumAPIVersionUnmet, version, minimum)
	}
	return nil
}

// V1 returns a client for version v1 of the Coordinator's HTTP API.
func (c *Client) V1() *apiv1.API {
	return apiv1.New(c.httpapi)
}
