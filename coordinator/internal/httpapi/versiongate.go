// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
)

// errAPIVersionTooOld rejects requests below the manifest's pinned minimum API version.
var errAPIVersionTooOld = errors.New("API version rejected by the manifest")

// APIVersionGate wraps an HTTP API endpoint and enforces the minimum API version optionally
// pinned in the deployment's manifest.
//
// Clients enforce the pin on their side, but without the gate an attacker who can tamper with
// client traffic could still talk to the Coordinator over an older API version.
type APIVersionGate struct {
	// Version is the API version of the wrapped endpoint. It is 0 for the legacy endpoints
	// that predate API versioning, which sort below all versioned endpoints.
	Version int
	// StateGuard provides the manifest whose pin is enforced.
	StateGuard StateGuard
	// Next is the wrapped endpoint.
	Next http.Handler
}

// ServeHTTP implements [http.Handler].
func (g *APIVersionGate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	state, err := g.StateGuard.GetState(r.Context())
	switch {
	case errors.Is(err, stateguard.ErrNoState), errors.Is(err, stateguard.ErrStaleState):
		// Without an unsealed manifest there is no pin to enforce. The initial SetManifest must pass while there is no manifest yet.
		g.Next.ServeHTTP(w, r)
		return
	case err != nil:
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("%w: %w", errGettingState, err))
		return
	}

	pin := state.Manifest().MinimumAPIVersion
	if pin == "" {
		g.Next.ServeHTTP(w, r)
		return
	}
	minVersion, err := apitypes.ParseAPIVersion(pin)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("parsing the manifest's MinimumAPIVersion: %w", err))
		return
	}
	if g.Version < minVersion {
		requested := fmt.Sprintf("version v%d", g.Version)
		if g.Version == 0 {
			requested = "the unversioned legacy API"
		}
		writeJSONError(w, http.StatusForbidden,
			fmt.Errorf("%w: the manifest requires at least API version %s, but the request used %s", errAPIVersionTooOld, pin, requested))
		return
	}
	g.Next.ServeHTTP(w, r)
}
