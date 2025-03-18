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
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/seedengine"
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

// fetchState creates a fresh state from the history that's verified by the given SeedEngine.
func (m *Authority) fetchState(se *seedengine.SeedEngine) (*State, error) {
	latest, err := m.hist.GetLatest(&se.TransactionSigningKey().PublicKey)
	if err != nil {
		return nil, fmt.Errorf("getting latest transition: %w", err)
	}

	// The latest transition in the backend is newer than ours, so we need to update our state.

	transition, err := m.hist.GetTransition(latest.TransitionHash)
	if err != nil {
		return nil, fmt.Errorf("getting transition: %w", err)
	}

	manifestBytes, err := m.hist.GetManifest(transition.ManifestHash)
	if err != nil {
		return nil, fmt.Errorf("getting manifest: %w", err)
	}
	mnfst := &manifest.Manifest{}
	if err := json.Unmarshal(manifestBytes, mnfst); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	meshKey, err := se.GenerateMeshCAKey()
	if err != nil {
		return nil, fmt.Errorf("deriving mesh CA key: %w", err)
	}
	ca, err := ca.New(se.RootCAKey(), meshKey)
	if err != nil {
		return nil, fmt.Errorf("creating CA: %w", err)
	}
	var generation int
	err = m.walkTransitions(latest.TransitionHash, func(_ [history.HashSize]byte, _ *history.Transition) error {
		generation++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking transitions: %w", err)
	}
	nextState := &State{
		seedEngine:    se,
		latest:        latest,
		ca:            ca,
		manifest:      mnfst,
		manifestBytes: manifestBytes,
		generation:    generation,
	}
	return nextState, nil
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

// GetState syncs the current state and returns the loaded current state.
func (m *Authority) GetState() (*State, error) {
	state := m.state.Load()
	if state == nil {
		return nil, errors.New("coordinator is not initialized")
	} else if state.stale.Load() {
		// TODO(burgerdev): we could attempt peer recovery here.
		return nil, ErrNeedsRecovery
	}
	return state, nil
}

// State is a snapshot of the Coordinator's manifest history.
type State struct {
	seedEngine    *seedengine.SeedEngine
	manifest      *manifest.Manifest
	manifestBytes []byte
	ca            *ca.CA

	latest     *history.LatestTransition
	generation int

	// stale is set to true when we discover that this State is not current anymore.
	// This field is only ever flipped from false to true!
	stale atomic.Bool
}

// NewState constructs a new State object.
//
// This function is intended for other packages that work on State objects. It does not produce a
// State that is valid for this package.
func NewState(seedEngine *seedengine.SeedEngine, manifest *manifest.Manifest, manifestBytes []byte, ca *ca.CA) *State {
	return &State{
		seedEngine:    seedEngine,
		manifest:      manifest,
		manifestBytes: manifestBytes,
		ca:            ca,
	}
}

// SeedEngine returns the SeedEngine for this state.
func (s *State) SeedEngine() *seedengine.SeedEngine {
	return s.seedEngine
}

// Manifest returns the Manifest for this state. It must not be modified.
func (s *State) Manifest() *manifest.Manifest {
	// TODO(burgerdev): consider deep-copying for safety
	return s.manifest
}

// ManifestBytes returns the raw bytes of the manifest for this state.
func (s *State) ManifestBytes() []byte {
	return s.manifestBytes
}

// CA returns the CA for this state.
func (s *State) CA() *ca.CA {
	return s.ca
}
