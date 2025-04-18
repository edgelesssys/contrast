// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// Package authority guards the current state of the Coordinator.
//
// The authority.Authority struct is the single source of truth for the currently enforced manifest
// and related data (secrets, metrics). It manages an authority.State object that can be handed
// out to other packages that need to operate on a current state. A State object does not change
// and can be safely used as long as necessary, but callers need to ensure that they consistently
// operate on a single state. For example, gRPC calls authenticated with the Authority.Credentials
// must only work with the state object added to the request context.
// The Authority exposes methods to manipulate the state by either updating to a new manifest or
// resetting to a manifest that was once active. To make these manipulations consistent, the
// method signatures resemble a compare-and-swap operation: callers need to first get the current
// state, then decide based on that state whether/how to change the state, and finally call the
// state manipulation method with the old state and the desired next state.
// For scenarios with multiple Coordinator instances, the Authority exposes a watcher routine that
// keeps track of changes to the persistency and marks the current state as stale if something else
// persisted changes. The state is also marked as stale if an inconsistency is discovered during
// state manipulation. A stale state can still be operated upon, since it was valid at some point,
// but recovery needs to happen before this Coordinator instance can update the state again.
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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/utils/clock"
)

var (
	// ErrNoManifest is returned when a manifest is needed but not present.
	ErrNoManifest = errors.New("no manifest configured")

	// ErrNoState is returned by GetState if the Coordinator has no state.
	ErrNoState = errors.New("coordinator is not initialized")

	// ErrConcurrentUpdate is returned by state-modifying operations if the input oldState is not
	// the current state. This usually happens when a concurrent operation succeeded.
	ErrConcurrentUpdate = errors.New("coordinator state was updated concurrently")
)

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
				walkErr := m.hist.WalkTransitions(state.latest.TransitionHash, func(h [32]byte, _ *history.Transition) error {
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
			m.logger.Info("time for a new watcher")
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ResetState changes the current state to reflect the given latest transition.
//
// This function is intended to recover to a state that is already present in the history. It does
// not verify the authenticity of the transition and does not write anything to the history. It's
// the callers responsibility to ensure the provided transition is present in history and that the
// secrets are valid for that transition. In particular, the mesh authority key needs to be either
// generated freshly (if recovering from scratch) or be received from an authenticated Coordinator
// peer operating on the same state.
func (m *Authority) ResetState(oldState *State, se *seedengine.SeedEngine, latest *history.LatestTransition, meshCAKey *ecdsa.PrivateKey) (*State, error) {
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

	ca, err := ca.New(se.RootCAKey(), meshCAKey)
	if err != nil {
		return nil, fmt.Errorf("creating CA: %w", err)
	}
	var generation int
	err = m.hist.WalkTransitions(latest.TransitionHash, func(_ [history.HashSize]byte, _ *history.Transition) error {
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
	if !m.state.CompareAndSwap(oldState, nextState) {
		return nil, ErrConcurrentUpdate
	}
	return nextState, nil
}

// GetState either returns the current state, or ErrNoState if the authority is not initialized.
func (m *Authority) GetState() (*State, error) {
	// TODO(burgerdev): should freshness be checked here?
	state := m.state.Load()
	if state == nil {
		return nil, ErrNoState
	}
	return state, nil
}

// UpdateState advances the Coordinator state to a new manifest generation.
//
// The oldState argument needs to be a state obtained from GetState. If the Coordinator state
// changes between the calls to GetState and UpdateState, an ErrConcurrentUpdate is returned.
func (m *Authority) UpdateState(oldState *State, se *seedengine.SeedEngine, manifestBytes []byte, meshCAKey *ecdsa.PrivateKey) (*State, error) {
	var mnfst manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &mnfst); err != nil {
		return nil, fmt.Errorf("unmarshaling manifest: %w", err)
	}
	manifestHash, err := m.hist.SetManifest(manifestBytes)
	if err != nil {
		return nil, fmt.Errorf("storing manifest: %w", err)
	}
	transition := &history.Transition{
		ManifestHash: manifestHash,
	}
	var oldLatest *history.LatestTransition
	var oldGeneration int
	if oldState != nil {
		transition.PreviousTransitionHash = oldState.latest.TransitionHash
		oldLatest = oldState.latest
		oldGeneration = oldState.generation
	}
	transitionHash, err := m.hist.SetTransition(transition)
	if err != nil {
		return nil, fmt.Errorf("storing transition: %w", err)
	}
	latest := &history.LatestTransition{
		TransitionHash: transitionHash,
	}
	if err := m.hist.SetLatest(oldLatest, latest, se.TransactionSigningKey()); err != nil {
		// TODO(burgerdev): check returned error, set state stale if it's a CAS failure and return ErrConcurrentUpdate.
		return nil, fmt.Errorf("updating latest transition: %w", err)
	}

	ca, err := ca.New(se.RootCAKey(), meshCAKey)
	if err != nil {
		return nil, fmt.Errorf("creating CA: %w", err)
	}

	nextState := &State{
		seedEngine:    se,
		manifest:      &mnfst,
		manifestBytes: manifestBytes,
		ca:            ca,
		latest:        latest,
		generation:    oldGeneration + 1,
	}
	m.state.CompareAndSwap(oldState, nextState)
	// If the CompareAndSwap did not go through, this means that another UpdateState happened in
	// the meantime. This is fine: we know that m.state must be a transition after ours because
	// the SetLatest call succeeded. That other UpdateState call must have been operating on our
	// nextState already, because it had to refer to our transition. Thus, we can ignore the return
	// value of CompareAndSwap, but we need to return the (now intermediate) state that the caller
	// intended and we must not unconditionally force our state as current.
	return nextState, nil
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

func (s *State) LatestTransitionHash() [history.HashSize]byte {
	return s.latest.TransitionHash
}

func (s *State) IsStale() bool {
	return s.stale.Load()
}

// ErrNeedsRecovery is returned if state exists, but no secrets are available, e.g. after restart.
var ErrNeedsRecovery = errors.New("coordinator is in recovery mode")
