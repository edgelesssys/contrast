// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"net/http"

	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/internal/atls"
)

// NewMux returns the handler serving all endpoints of the Coordinator's HTTP API.
func NewMux(issuer atls.Issuer, guard StateGuard) *http.ServeMux {
	capabilities := NewCapabilitiesHandler()

	mux := http.NewServeMux()
	mux.Handle("/capabilities", capabilities)
	// Legacy endpoint, from before the API was versioned. Kept so that older clients keep working.
	mux.Handle(apitypesv1.LegacyAttestPath, &APIVersionGate{Version: 0, StateGuard: guard, Next: &AttestationHandler{
		Issuer:     issuer,
		StateGuard: guard,
	}})
	return mux
}
