// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

// Package apiv1 implements version v1 of the Contrast HTTP API.
//
// Obtain an [API] from the SDK client via its V1 method, rather than constructing one directly.
// Requests and responses differ between API versions, so the SDK exposes each version through a package of its own.
// Callers that support several versions negotiate one with the SDK client's NegotiateAPIVersion method and use
// the matching package.
package apiv1

import (
	"github.com/edgelesssys/contrast/apitypes"
	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/atls/validators"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/sdk/internal/httpapi"
)

// Version is the API version implemented by this package.
const Version = apitypesv1.Version

// Config is the configuration an [API] shares with the SDK client it was obtained from.
type Config struct {
	// HTTP issues the requests to the Coordinator.
	HTTP *httpapi.Client
	// Validator returns the validator for the attestation of a Coordinator running the given manifest.
	Validator func(*manifest.Manifest) (validators.Validator, error)
	// MinimumAPIVersion returns the optional minimum API version pinned by the caller.
	//
	// It is called by every operation, so that later changes to the SDK client's pin apply.
	MinimumAPIVersion func() string
	// NegotiatedCapabilities returns the capabilities the SDK client negotiated the API version with,
	// or nil if it didn't negotiate.
	NegotiatedCapabilities func() *apitypes.CapabilitiesResponse
}

// API calls version v1 of the Coordinator's HTTP API.
type API struct {
	cfg Config
}

// New returns an [API] with the given configuration.
func New(cfg Config) *API {
	return &API{cfg: cfg}
}

// minimumAPIVersion returns the minimum API version currently pinned by the caller, or an empty string if there is none.
func (a *API) minimumAPIVersion() string {
	if a.cfg.MinimumAPIVersion == nil {
		return ""
	}
	return a.cfg.MinimumAPIVersion()
}
