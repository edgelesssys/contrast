// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/coordinator/internal/seedengine"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/crypto"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ErrNoManifest is returned when a manifest is needed but not present.
var ErrNoManifest = errors.New("no manifest configured")

// Bundle is a set of PEM-encoded certificates for Contrast workloads.
type Bundle struct {
	WorkloadCert   []byte
	MeshCA         []byte
	IntermediateCA []byte
	RootCA         []byte
}

// Authority manages the manifest state of Contrast.
type Authority struct {
	// state holds all required configuration to serve requests from userapi.
	// We bundle it in its own struct type so we can atomically update it while not blocking other
	// requests or risking inconsistency between e.g. CA and Manifest.
	state      atomic.Pointer[state]
	se         atomic.Pointer[seedengine.SeedEngine]
	hist       *history.History
	bundles    map[string]Bundle
	bundlesMux sync.RWMutex
	logger     *slog.Logger
	metrics    metrics
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
		bundles: make(map[string]Bundle),
		hist:    hist,
		logger:  log.WithGroup("mesh-authority"),
		metrics: metrics{
			manifestGeneration: manifestGeneration,
		},
	}
}

// SNPValidateOpts returns SNP validation options from reference values.
//
// It also ensures that the policy hash in the report's HOSTDATA is allowed by the current
// manifest.
func (m *Authority) SNPValidateOpts(report *sevsnp.Report) (*validate.Options, error) {
	state := m.state.Load()
	if state == nil {
		return nil, ErrNoManifest
	}
	mnfst := state.manifest

	hostData := manifest.NewHexString(report.HostData)
	if _, ok := mnfst.Policies[hostData]; !ok {
		return nil, fmt.Errorf("hostdata %s not found in manifest", hostData)
	}

	trustedMeasurement, err := mnfst.ReferenceValues.TrustedMeasurement.Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to convert TrustedMeasurement from manifest to byte slices: %w", err)
	}
	if trustedMeasurement == nil {
		// This is required to prevent an empty measurement in the manifest from disabling the measurement check.
		trustedMeasurement = make([]byte, 48)
	}

	return &validate.Options{
		Measurement: trustedMeasurement,
		GuestPolicy: abi.SnpPolicy{
			Debug: false,
			SMT:   true,
		},
		VMPL: new(int), // VMPL0
		MinimumTCB: kds.TCBParts{
			BlSpl:    mnfst.ReferenceValues.SNP.MinimumTCB.BootloaderVersion.UInt8(),
			TeeSpl:   mnfst.ReferenceValues.SNP.MinimumTCB.TEEVersion.UInt8(),
			SnpSpl:   mnfst.ReferenceValues.SNP.MinimumTCB.SNPVersion.UInt8(),
			UcodeSpl: mnfst.ReferenceValues.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
		},
		MinimumLaunchTCB: kds.TCBParts{
			BlSpl:    mnfst.ReferenceValues.SNP.MinimumTCB.BootloaderVersion.UInt8(),
			TeeSpl:   mnfst.ReferenceValues.SNP.MinimumTCB.TEEVersion.UInt8(),
			SnpSpl:   mnfst.ReferenceValues.SNP.MinimumTCB.SNPVersion.UInt8(),
			UcodeSpl: mnfst.ReferenceValues.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
		},
		PermitProvisionalFirmware: true,
	}, nil
}

// ValidateCallback creates a certificate bundle for the verified client.
func (m *Authority) ValidateCallback(_ context.Context, report *sevsnp.Report,
	_ asn1.ObjectIdentifier, _, _, peerPubKeyBytes []byte,
) error {
	state := m.state.Load()
	if state == nil {
		return ErrNoManifest
	}

	hostData := manifest.NewHexString(report.HostData)
	dnsNames, ok := state.manifest.Policies[hostData]
	if !ok {
		return fmt.Errorf("report data %s not found in manifest", hostData)
	}

	peerPubKey, err := x509.ParsePKIXPublicKey(peerPubKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse peer public key: %w", err)
	}

	extensions, err := snp.ClaimsToCertExtension(report)
	if err != nil {
		return fmt.Errorf("failed to construct extensions: %w", err)
	}
	cert, err := state.ca.NewAttestedMeshCert(dnsNames, extensions, peerPubKey)
	if err != nil {
		return fmt.Errorf("failed to issue new attested mesh cert: %w", err)
	}

	peerPubKeyHash := sha256.Sum256(peerPubKeyBytes)
	peerPublicKeyHashStr := hex.EncodeToString(peerPubKeyHash[:])
	m.logger.Info("Validated peer", "peerPublicKeyHashStr", peerPublicKeyHashStr)

	m.bundlesMux.Lock()
	defer m.bundlesMux.Unlock()
	m.bundles[peerPublicKeyHashStr] = Bundle{
		WorkloadCert:   cert,
		MeshCA:         state.ca.GetMeshCACert(),
		IntermediateCA: state.ca.GetIntermCACert(),
		RootCA:         state.ca.GetRootCACert(),
	}

	return nil
}

// GetCertBundle retrieves the certificate bundle created for the peer identified by the given public key.
func (m *Authority) GetCertBundle(peerPublicKeyHashStr string) (Bundle, error) {
	m.bundlesMux.RLock()
	defer m.bundlesMux.RUnlock()

	bundle, ok := m.bundles[peerPublicKeyHashStr]

	if !ok {
		return Bundle{}, fmt.Errorf("cert for peer public key hash %s not found", peerPublicKeyHashStr)
	}

	return bundle, nil
}

// GetManifestsAndLatestCA retrieves the manifest history and the currently active CA instance.
//
// If no manifest is configured, it returns an empty slice and a nil CA.
func (m *Authority) GetManifestsAndLatestCA() ([]*manifest.Manifest, *ca.CA, error) {
	if m.se.Load() == nil {
		return nil, nil, ErrNoManifest
	}
	if err := m.syncState(); err != nil {
		return nil, nil, fmt.Errorf("syncing internal state: %w", err)
	}
	state := m.state.Load()
	if state == nil {
		return nil, nil, ErrNoManifest
	}

	var manifests []*manifest.Manifest
	err := m.walkTransitions(state.latest.TransitionHash, func(_ [history.HashSize]byte, t *history.Transition) error {
		manifestBytes, err := m.hist.GetManifest(t.ManifestHash)
		if err != nil {
			return err
		}
		var mnfst manifest.Manifest
		if err := json.Unmarshal(manifestBytes, &mnfst); err != nil {
			return err
		}
		manifests = append(manifests, &mnfst)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	// Traversing the history yields manifests in the wrong order, so reverse the slice.
	slices.Reverse(manifests)
	return manifests, state.ca, nil
}

// SetManifest updates the active manifest.
func (m *Authority) SetManifest(manifestBytes []byte, policies [][]byte) (*ca.CA, error) {
	if err := m.createSeedEngine(); err != nil {
		return nil, fmt.Errorf("creating SeedEngine: %w", err)
	}

	if err := m.syncState(); err != nil {
		return nil, fmt.Errorf("syncing internal state: %w", err)
	}

	var mnfst manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &mnfst); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
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
			return nil, fmt.Errorf("no policy for hash %q", hexRef)
		}
	}

	manifestHash, err := m.hist.SetManifest(manifestBytes)
	if err != nil {
		return nil, fmt.Errorf("setting manifest: %w", err)
	}

	oldState := m.state.Load()

	nextTransition := &history.Transition{
		ManifestHash: manifestHash,
	}
	var oldLatest *history.LatestTransition
	var oldGeneration int
	if oldState != nil {
		nextTransition.PreviousTransitionHash = oldState.latest.TransitionHash
		oldLatest = oldState.latest
		oldGeneration = oldState.generation
	}
	nextTransitionHash, err := m.hist.SetTransition(nextTransition)
	if err != nil {
		return nil, fmt.Errorf("setting transition: %w", err)
	}
	nextLatest := &history.LatestTransition{TransitionHash: nextTransitionHash}

	if err := m.hist.SetLatest(oldLatest, nextLatest); err != nil {
		return nil, fmt.Errorf("setting latest: %w", err)
	}

	meshKey, err := m.se.Load().DeriveMeshCAKey(nextTransitionHash)
	if err != nil {
		return nil, fmt.Errorf("deriving mesh CA key: %w", err)
	}
	ca, err := ca.New(m.se.Load().RootCAKey(), meshKey)
	if err != nil {
		return nil, fmt.Errorf("creating CA: %w", err)
	}

	nextState := &state{
		latest:     nextLatest,
		manifest:   &mnfst,
		ca:         ca,
		generation: oldGeneration + 1,
	}

	if m.state.CompareAndSwap(oldState, nextState) {
		m.metrics.manifestGeneration.Set(float64(nextState.generation))
	}
	// If the CompareAndSwap did not go through, this means that another SetManifest happened in
	// the meantime. This is fine: we know that m.state must be a transition after ours because
	// the SetLatest call succeeded. That other SetManifest call must have been operating on our
	// nextState already, because it had to refer to our transition. Thus, we can forget about
	// the state, except that we need to return the right CA for the manifest _our_ user set.

	return ca, nil
}

// LatestManifest retrieves the active manifest.
func (m *Authority) LatestManifest() (*manifest.Manifest, error) {
	if m.se.Load() == nil {
		return nil, ErrNoManifest
	}
	if err := m.syncState(); err != nil {
		return nil, fmt.Errorf("syncing internal state: %w", err)
	}
	c := m.state.Load()
	if c == nil {
		return nil, ErrNoManifest
	}
	return c.manifest, nil
}

// createSeedEngine populates m.se.
//
// It is fine to call this function concurrently. After it returns, m.se is guaranteed to be
// non-nil.
func (m *Authority) createSeedEngine() error {
	// TODO(burgerdev): the seed should be an input
	seed, err := crypto.GenerateRandomBytes(32)
	if err != nil {
		return fmt.Errorf("generating random bytes: %w", err)
	}
	salt, err := crypto.GenerateRandomBytes(32)
	if err != nil {
		return fmt.Errorf("generating random bytes: %w", err)
	}
	seedEngine, err := seedengine.New(seed, salt)
	if err != nil {
		return fmt.Errorf("creating seed engine: %w", err)
	}
	// It's fine if the seedEngine has already been created by another thread.
	m.se.CompareAndSwap(nil, seedEngine)

	m.hist.ConfigureSigningKey(m.se.Load().TransactionSigningKey())
	return nil
}

// syncState ensures that a.state is up-to-date.
//
// This function guarantees to include all state updates committed before it was called.
func (m *Authority) syncState() error {
	oldState := m.state.Load()
	latest, err := m.hist.GetLatest()
	if errors.Is(err, os.ErrNotExist) {
		return nil // No history yet -> nothing to sync.
	} else if err != nil {
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

	meshKey, err := m.se.Load().DeriveMeshCAKey(latest.TransitionHash)
	if err != nil {
		return fmt.Errorf("deriving mesh CA key: %w", err)
	}
	ca, err := ca.New(m.se.Load().RootCAKey(), meshKey)
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
	nextState := &state{
		latest:     latest,
		ca:         ca,
		manifest:   mnfst,
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

type state struct {
	latest     *history.LatestTransition
	manifest   *manifest.Manifest
	ca         *ca.CA
	generation int
}
