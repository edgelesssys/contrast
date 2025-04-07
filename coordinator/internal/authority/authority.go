// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/seedengine"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/utils/clock"
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

	clock clock.Clock

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
		clock: clock.RealClock{},
	}
}

// WatchHistory monitors the history for manifest updates and sets the state stale if necessary.
//
// This function blocks and keeps watching until the context expires.
func (m *Authority) WatchHistory(ctx context.Context) error {
	for {
		transitionUpdates, err := m.hist.WatchLatestTransitions(ctx)
	loop:
		for err == nil {
			select {
			case t, ok := <-transitionUpdates:
				if !ok {
					err = fmt.Errorf("channel closed unexpectedly")
					break loop
				}
				// We received a new latest transition, so check whether the update is already
				// reflected in the state. If this Coordinator instance did the update, there is a
				// race condition between the watcher notification and the state being replaced by
				// SetManifest. This is not a problem:
				//   - If we get the old state, we mark it stale but it's going to be replaced
				//     anyway.
				//   - If we get the new state, we see the matching transition hashes and don't
				//     mark the state stale.
				// There's a theoretically problematic race when the manifest is updated twice in
				// quick succession on this Coordinator, and the watcher notifications arrive late.
				// In that situation, we would be marking a state as stale that is actually fresh.
				// Since we were the Coordinator doing the state update, we're likely the only one
				// that has the new mesh certificate, and thus going stale would mean losing the
				// ability to recover the cluster automatically. Thus, we check whether the state
				// we were notified about is an ancestor of the current state.
				state := m.state.Load()
				if state == nil {
					continue
				}
				stateInAncestors := false
				walkErr := m.walkTransitions(state.latest.TransitionHash, func(h [32]byte, _ *history.Transition) error {
					if h == t.TransitionHash {
						stateInAncestors = true
					}
					return nil
				})
				if walkErr != nil {
					m.logger.Warn("received problematic transition update", "error", err)
					continue
				}
				if stateInAncestors {
					continue
				}
				m.logger.Info("History changed, marking state as stale",
					"from-transition", manifest.NewHexString(state.latest.TransitionHash[:]),
					"to-transition", manifest.NewHexString(t.TransitionHash[:]))
				state.stale.Store(true)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		m.logger.Error("WatchLatestTransitions failed, starting a new watcher", "error", err)
		select {
		case <-m.clock.After(5 * time.Second):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// RecoverWith recovers the Coordinator to the given transition.
//
// The legitimacy of the transition must be checked by the caller. This call fails if the
// Coordinator does not require recovery.
func (m *Authority) RecoverWith(se *seedengine.SeedEngine, latest *history.LatestTransition, meshKey *ecdsa.PrivateKey) error {
	oldState := m.state.Load()
	if oldState != nil && !oldState.stale.Load() {
		return ErrAlreadyRecovered
	}

	state, err := m.fetchState(se, latest, meshKey)
	if err != nil {
		return fmt.Errorf("fetching state: %w", err)
	}

	if !m.state.CompareAndSwap(oldState, state) {
		return ErrConcurrentRecovery
	}
	return nil
}

// fetchState creates a fresh state from the given transition.
//
// fetchState does not do any verification of the transition.
func (m *Authority) fetchState(se *seedengine.SeedEngine, latest *history.LatestTransition, meshKey *ecdsa.PrivateKey) (*State, error) {
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
