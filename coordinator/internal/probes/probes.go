// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package probes

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
)

// StartupHandler is the http handler for `/probes/startup`.
type StartupHandler struct {
	UserapiStarted *atomic.Bool
	MeshapiStarted *atomic.Bool
}

func (h StartupHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if !h.UserapiStarted.Load() || !h.MeshapiStarted.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	// TODO(miampf): Check if peer recovery was attempted once
	w.WriteHeader(http.StatusOK)
}

// ReadinessHandler is the http handler for `/probes/readiness`.
type ReadinessHandler struct {
	Guard guard
}

func (h ReadinessHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	state, err := h.Guard.GetState(req.Context())
	if err != nil || state == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type guard interface {
	GetState(context.Context) (*stateguard.State, error)
}
