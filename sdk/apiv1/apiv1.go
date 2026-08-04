// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

// Package apiv1 implements version v1 of the Contrast HTTP API.
//
// Obtain an [API] from the SDK client via its V1 method, rather than constructing one
// directly. Use this package to pin calls to v1; the SDK's top-level methods always speak
// the newest API version the Coordinator supports.
package apiv1

import (
	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/sdk/internal/httpapi"
)

// Version is the API version implemented by this package.
const Version = apitypes.APIVersionV1

// API calls version v1 of the Coordinator's HTTP API.
type API struct {
	httpapi *httpapi.Client
}

// New returns an [API] issuing its requests through the given HTTP API client.
func New(c *httpapi.Client) *API {
	return &API{httpapi: c}
}
