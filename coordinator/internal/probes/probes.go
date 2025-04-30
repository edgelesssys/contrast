// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package probes

import (
	"net/http"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
)

// StartupHandler is the http handler for `/probes/startup`.
type StartupHandler struct {
	UserapiStarted bool
	MeshapiStarted bool
}

func (h StartupHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if !h.UserapiStarted || !h.MeshapiStarted {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	// TODO(miampf): Check if peer recovery was attempted once
	w.WriteHeader(http.StatusOK)
}

// LivenessHandler is the http handler for `/probes/liveness`.
type LivenessHandler struct {
	Hist *history.History
}

func (h LivenessHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	_, err := h.Hist.HasLatest()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ReadinessHandler is the http handler for `/probes/readiness`.
type ReadinessHandler struct {
	Guard guard
}

func (h ReadinessHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	state, err := h.Guard.GetState()
	if err != nil || state == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	// TODO(miampf): Check that coordinator isn't in recovery mode
	w.WriteHeader(http.StatusOK)
}

type guard interface {
	GetState() (*stateguard.State, error)
}
