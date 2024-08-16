// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/coordinator/internal/seedengine"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ErrNoManifest is returned when a manifest is needed but not present.
var ErrNoManifest = errors.New("no manifest configured")

// Authority manages the manifest state of Contrast.
type Authority struct {
	// state holds all required configuration to serve requests from userapi.
	// We bundle it in its own struct type so we can atomically update it while not blocking other
	// requests or risking inconsistency between e.g. CA and Manifest.
	// state must always be updated with a new instance, its fields must not be modified.
	state   atomic.Pointer[State]
	se      atomic.Pointer[seedengine.SeedEngine]
	hist    *history.History
	logger  *slog.Logger
	metrics metrics

	userapi.UnimplementedUserAPIServer
}

type metrics struct {
	manifestGeneration prometheus.Gauge
}

// New creates a new Authority instance.
func New(hist *history.History, reg *prometheus.Registry, log *slog.Logger) *Authority {
	manifestGeneration := promauto.With(reg).NewGauge(prometheus.GaugeOpts{
		Subsystem: "contrast_coordinator",
		Name:      "manifest_generation",
		Help:      "Current manifest generation.",
	})
	manifestGeneration.Set(0)

	return &Authority{
		hist:   hist,
		logger: log.WithGroup("mesh-authority"),
		metrics: metrics{
			manifestGeneration: manifestGeneration,
		},
	}
}

// initSeedEngine recovers the seed engine from a seed and salt.
func (m *Authority) initSeedEngine(seed, salt []byte) error {
	seedEngine, err := seedengine.New(seed, salt)
	if err != nil {
		return fmt.Errorf("creating seed engine: %w", err)
	}
	if !m.se.CompareAndSwap(nil, seedEngine) {
		return ErrAlreadyRecovered
	}
	m.hist.ConfigureSigningKey(m.se.Load().TransactionSigningKey())
	return nil
}

// syncState ensures that a.state is up-to-date.
//
// This function guarantees to include all state updates committed before it was called.
func (m *Authority) syncState() error {
	hasLatest, err := m.hist.HasLatest()
	if err != nil {
		return fmt.Errorf("probing latest transition: %w", err)
	}
	if !hasLatest {
		// No history yet -> nothing to sync.
		return nil
	}
	se := m.se.Load()
	if se == nil {
		// There are transitions in history, but we don't have a signing key -> recovery mode.
		return ErrNeedsRecovery
	}

	oldState := m.state.Load()
	latest, err := m.hist.GetLatest()
	if err != nil {
		return fmt.Errorf("getting latest transition: %w", err)
	}
	if oldState != nil && latest.TransitionHash == oldState.latest.TransitionHash {
		return nil
	}

	// The latest transition in the backend is newer than ours, so we need to update our state.

	transition, err := m.hist.GetTransition(latest.TransitionHash)
	if err != nil {
		return fmt.Errorf("getting transition: %w", err)
	}

	manifestBytes, err := m.hist.GetManifest(transition.ManifestHash)
	if err != nil {
		return fmt.Errorf("getting manifest: %w", err)
	}
	mnfst := &manifest.Manifest{}
	if err := json.Unmarshal(manifestBytes, mnfst); err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	meshKey, err := se.GenerateMeshCAKey()
	if err != nil {
		return fmt.Errorf("deriving mesh CA key: %w", err)
	}
	ca, err := ca.New(se.RootCAKey(), meshKey)
	if err != nil {
		return fmt.Errorf("creating CA: %w", err)
	}
	var generation int
	err = m.walkTransitions(latest.TransitionHash, func(_ [history.HashSize]byte, _ *history.Transition) error {
		generation++
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking transitions: %w", err)
	}
	nextState := &State{
		latest:     latest,
		CA:         ca,
		Manifest:   mnfst,
		generation: generation,
	}
	// We consider the sync successful even if the state was updated by someone else.
	if m.state.CompareAndSwap(oldState, nextState) {
		// Only set the gauge if our state modification was actually successful - otherwise, it
		// won't match the active state.
		m.metrics.manifestGeneration.Set(float64(generation))
	}
	return nil
}

// walkTransitions executes a function for all transitions in the history of transitionRef, starting from most recent.
func (m *Authority) walkTransitions(transitionRef [history.HashSize]byte, consume func([history.HashSize]byte, *history.Transition) error) error {
	for transitionRef != [history.HashSize]byte{} {
		transition, err := m.hist.GetTransition(transitionRef)
		if err != nil {
			return fmt.Errorf("getting transition: %w", err)
		}
		if err := consume(transitionRef, transition); err != nil {
			return err
		}
		transitionRef = transition.PreviousTransitionHash
	}
	return nil
}

// GetSeedEngine returns the seed engine.
func (m *Authority) GetSeedEngine() (*seedengine.SeedEngine, error) {
	se := m.se.Load()
	if se == nil {
		return nil, errors.New("seed engine not initialized")
	}
	return se, nil
}

// State is a snapshot of the Coordinator's manifest history.
type State struct {
	Manifest *manifest.Manifest
	CA       *ca.CA

	latest     *history.LatestTransition
	generation int
}
