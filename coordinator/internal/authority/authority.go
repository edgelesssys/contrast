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
	"slices"
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
	// ErrNoState is returned by GetState if the Coordinator has no state.
	ErrNoState = errors.New("coordinator is not initialized")

	// ErrStaleState is returned if state exists but is stale.
	ErrStaleState = errors.New("coordinator state is outdated")

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

// ResetState resets the Coordinator state to the latest persisted state.
//
// This function is intended to be called in recovery scenarios, where a latest state exists but
// can't be verified without the necessary secrets. The state is recovered by first loading the
// state insecurely, then passing the corresponding manifest to the caller-provided authorization
// function, and finally using the key material to verify the loaded state.
//
// The authorizeSeedSource function must check that the source of the seed (a user or a peer
// Coordinator) are authorized to hold the secret seed, according to the manifest that's being
// recovered to. See RFC 010 for more details on the security considerations for handling seeds.
func (m *Authority) ResetState(oldState *State, authorizer SecretSourceAuthorizer) (*State, error) {
	insecureLatest, err := m.hist.GetLatestInsecure()
	if err != nil {
		return nil, fmt.Errorf("getting latest transition: %w", err)
	}

	transition, err := m.hist.GetTransition(insecureLatest.TransitionHash)
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

	se, meshCAKey, err := authorizer.AuthorizeByManifest(mnfst)
	if err != nil {
		return nil, fmt.Errorf("authorizing seed source: %w", err)
	}

	latest, err := m.hist.GetLatest(&se.TransactionSigningKey().PublicKey)
	if err != nil {
		return nil, fmt.Errorf("verifying latest transition: %w", err)
	}

	if insecureLatest.TransitionHash != latest.TransitionHash {
		return nil, fmt.Errorf("%w: transition changed from %x to %x", ErrConcurrentUpdate, insecureLatest.TransitionHash, latest.TransitionHash)
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
	m.metrics.manifestGeneration.Set(float64(generation))
	return nextState, nil
}

// SecretSourceAuthorizer obtains secrets and authorizes their source.
type SecretSourceAuthorizer interface {
	// AuthorizeByManifest obtains a SeedEngine and a mesh CA key and verifies their source
	// according to the Manifest. Secrets must only be held by other Coordinators (identified by
	// their Role) and seed share owners.
	AuthorizeByManifest(*manifest.Manifest) (*seedengine.SeedEngine, *ecdsa.PrivateKey, error)
}

// GetState returns the current state.
//
// If no state is set, the returned state is nil and the error is ErrNoState.
// If a state is set but the latest state is newer, the state is returned and the error is ErrStaleState.
// If the state is up-to-date, the returned error is nil.
// The function may return a different error if the persistent state is not accessible.
func (m *Authority) GetState() (*State, error) {
	state := m.state.Load()
	if state == nil {
		hasLatest, err := m.hist.HasLatest()
		if err != nil {
			return nil, fmt.Errorf("checking state: %w", err)
		}
		if hasLatest {
			return nil, ErrStaleState
		}
		return nil, ErrNoState
	}
	if state.stale.Load() {
		return state, ErrStaleState
	}
	return state, nil
}

// UpdateState advances the Coordinator state to a new manifest generation.
//
// The oldState argument needs to be a state obtained from GetState. If the Coordinator state
// changes between the calls to GetState and UpdateState, an ErrConcurrentUpdate is returned.
func (m *Authority) UpdateState(oldState *State, se *seedengine.SeedEngine, manifestBytes []byte, policies [][]byte) (*State, error) {
	var mnfst manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &mnfst); err != nil {
		return nil, fmt.Errorf("unmarshaling manifest: %w", err)
	}
	policyMap := make(map[[history.HashSize]byte][]byte)
	for _, policy := range policies {
		policyHash, err := m.hist.SetPolicy(policy)
		if err != nil {
			return nil, fmt.Errorf("setting policy: %w", err)
		}
		policyMap[policyHash] = policy
	}

	for hexRef := range mnfst.Policies {
		var ref [history.HashSize]byte
		refSlice, err := hexRef.Bytes()
		if err != nil {
			return nil, fmt.Errorf("invalid policy hash: %w", err)
		}
		copy(ref[:], refSlice)
		if _, ok := policyMap[ref]; !ok {
			return nil, fmt.Errorf("no policy provided for hash %q", hexRef)
		}
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

	meshCAKey, err := se.GenerateMeshCAKey()
	if err != nil {
		return nil, fmt.Errorf("generating mesh CA key: %w", err)
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
	if !m.state.CompareAndSwap(oldState, nextState) {
		// If the CompareAndSwap did not go through, this means that another UpdateState happened in
		// the meantime. This is fine: we know that m.state must be a transition after ours because
		// the SetLatest call succeeded. That other UpdateState call must have been operating on our
		// nextState already, because it had to refer to our transition. Thus, we can ignore the return
		// value of CompareAndSwap, but we need to return the (now intermediate) state that the caller
		// intended and we must not unconditionally force our state as current or update the gauge.
		return nextState, nil
	}
	m.metrics.manifestGeneration.Set(float64(nextState.generation))
	return nextState, nil
}

// GetHistory returns a list of manifests, the current manifest being last, and the policies
// referenced in at least one of the manifests.
func (m *Authority) GetHistory() ([][]byte, map[manifest.HexString][]byte, error) {
	state, err := m.GetState()
	if err != nil {
		return nil, nil, err
	}
	var manifests [][]byte
	policies := make(map[manifest.HexString][]byte)
	err = m.hist.WalkTransitions(state.latest.TransitionHash, func(_ [history.HashSize]byte, t *history.Transition) error {
		manifestBytes, err := m.hist.GetManifest(t.ManifestHash)
		if err != nil {
			return err
		}
		manifests = append(manifests, manifestBytes)

		var mnfst manifest.Manifest
		if err := json.Unmarshal(manifestBytes, &mnfst); err != nil {
			return err
		}

		for policyHashHex := range mnfst.Policies {
			if _, ok := policies[policyHashHex]; ok {
				continue
			}
			policyHash, err := policyHashHex.Bytes()
			if err != nil {
				return fmt.Errorf("converting hex to bytes: %w", err)
			}
			var policyHashFixed [history.HashSize]byte
			copy(policyHashFixed[:], policyHash)
			policyBytes, err := m.hist.GetPolicy(policyHashFixed)
			if err != nil {
				return fmt.Errorf("getting policy: %w", err)
			}
			policies[policyHashHex] = policyBytes
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("fetching manifests from history: %w", err)
	}
	// Traversing the history yields manifests in the wrong order, so reverse the slice.
	slices.Reverse(manifests)

	return manifests, policies, nil
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

// NewStateForTest constructs a new State object.
//
// This function is intended for testing packages that work on State objects. It fills the fields
// that are observable outside this package, but does not manage the fields only relevant for this
// package. State objects created with this function can't be used as arguments to this package's
// public API functions.
func NewStateForTest(seedEngine *seedengine.SeedEngine, manifest *manifest.Manifest, manifestBytes []byte, ca *ca.CA) *State {
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
