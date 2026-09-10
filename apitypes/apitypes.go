// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package apitypes contains the wire-format types of the Contrast HTTP API.
//
// This package holds only what is independent of the API version.
// Everything that belongs to a specific version lives in a subpackage named after it, e.g.
// [github.com/edgelesssys/contrast/apitypes/apiv1] for the /v1/ endpoints.
package apitypes

// Port is the listening port of the HTTP API server.
const Port = "1314"
