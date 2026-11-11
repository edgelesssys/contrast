// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

// Package apiversion compares API versions against minimum version pins.
//
// It's shared by the SDK and its versioned API packages, which can't import the SDK.
package apiversion

import (
	"errors"
	"fmt"

	"github.com/edgelesssys/contrast/apitypes"
)

// Legacy is the version of the unversioned legacy API.
const Legacy = ""

// ErrMinimumUnmet is returned if the API version in use is older than the pinned minimum API version.
var ErrMinimumUnmet = errors.New("minimum API version not met")

// ErrDowngrade is returned if the API version was negotiated with capabilities that differ from the ones the Coordinator attested to.
var ErrDowngrade = errors.New("API version was downgraded")

// EnforceMinimum returns [ErrMinimumUnmet] if version is older than the optional minimum API version pin.
func EnforceMinimum(version string, minimum string) error {
	if minimum == "" {
		return nil
	}
	minVersion, err := apitypes.ParseAPIVersion(minimum)
	if err != nil {
		return fmt.Errorf("parsing the minimum API version: %w", err)
	}
	if version == Legacy {
		return fmt.Errorf("%w: the unversioned legacy API is older than the minimum %s required", ErrMinimumUnmet, minimum)
	}
	v, err := apitypes.ParseAPIVersion(version)
	if err != nil {
		return fmt.Errorf("parsing API version %q: %w", version, err)
	}
	if v < minVersion {
		return fmt.Errorf("%w: API version %s is older than the minimum %s required", ErrMinimumUnmet, version, minimum)
	}
	return nil
}
