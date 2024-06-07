// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/edgelesssys/contrast/coordinator/internal/seedengine"
	"github.com/edgelesssys/contrast/internal/appendable"
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

// Bundle is a set of PEM-encoded certificates for Contrast workloads.
type Bundle struct {
	WorkloadCert   []byte
	MeshCA         []byte
	IntermediateCA []byte
	RootCA         []byte
}

// Authority manages the manifest state of Contrast.
type Authority struct {
	se         atomic.Pointer[seedengine.SeedEngine]
	ca         atomic.Pointer[ca.CA]
	bundles    map[string]Bundle
	bundlesMux sync.RWMutex
	manifests  appendableList[*manifest.Manifest]
	logger     *slog.Logger
	metrics    metrics
}

type metrics struct {
	manifestGeneration prometheus.Gauge
}

// New creates a new Authority instance.
func New(reg *prometheus.Registry, log *slog.Logger) *Authority {
	manifestGeneration := promauto.With(reg).NewGauge(prometheus.GaugeOpts{
		Subsystem: "contrast_coordinator",
		Name:      "manifest_generation",
		Help:      "Current manifest generation.",
	})

	return &Authority{
		bundles:   make(map[string]Bundle),
		manifests: new(appendable.Appendable[*manifest.Manifest]),
		logger:    log.WithGroup("mesh-authority"),
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
	mnfst, err := m.manifests.Latest()
	if err != nil {
		return nil, fmt.Errorf("getting latest manifest: %w", err)
	}

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
	mnfst, err := m.manifests.Latest()
	if err != nil {
		return fmt.Errorf("getting latest manifest: %w", err)
	}
	// TODO(burgerdev): The CA should be tied to the manifest.
	caInstance := m.ca.Load()
	if caInstance == nil {
		return fmt.Errorf("no available CA")
	}

	hostData := manifest.NewHexString(report.HostData)
	dnsNames, ok := mnfst.Policies[hostData]
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
	cert, err := caInstance.NewAttestedMeshCert(dnsNames, extensions, peerPubKey)
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
		MeshCA:         caInstance.GetMeshCACert(),
		IntermediateCA: caInstance.GetIntermCACert(),
		RootCA:         caInstance.GetRootCACert(),
	}

	return nil
}

// GetCertBundle retrieves the certificate bundle created for the peer identified by the given public key.
func (m *Authority) GetCertBundle(peerPublicKeyHashStr string) (Bundle, error) {
	m.bundlesMux.RLock()
	defer m.bundlesMux.RUnlock()

	bundle, ok := m.bundles[peerPublicKeyHashStr]

	if !ok {
		return Bundle{}, fmt.Errorf("cert for peer public key %s not found", peerPublicKeyHashStr)
	}

	return bundle, nil
}

// GetManifestsAndLatestCA retrieves the manifest history and the currently active CA instance.
func (m *Authority) GetManifestsAndLatestCA() ([]*manifest.Manifest, *ca.CA) {
	// TODO(burgerdev): The CA should be tied to the manifest.
	return m.manifests.All(), m.ca.Load()
}

// SetManifest updates the active manifest.
func (m *Authority) SetManifest(mnfst *manifest.Manifest) error {
	se := m.se.Load()
	if se == nil {
		if err := m.createSeedEngine(); err != nil {
			return fmt.Errorf("could not create SeedEngine: %w", err)
		}
		se = m.se.Load()
	}
	// TODO(burgerdev): get hash from manifest transition
	hash, err := crypto.GenerateRandomBytes(32)
	if err != nil {
		return fmt.Errorf("generating random bytes: %w", err)
	}
	var fixedLengthHash [32]byte
	copy(fixedLengthHash[:], hash)
	meshKey, err := se.DeriveMeshCAKey(fixedLengthHash)
	if err != nil {
		return fmt.Errorf("deriving new mesh CA key: %w", err)
	}
	ca, err := ca.New(se.RootCAKey(), meshKey)
	if err != nil {
		return fmt.Errorf("creating new CA: %w", err)
	}
	m.ca.Store(ca)
	m.manifests.Append(mnfst)
	m.metrics.manifestGeneration.Set(float64(len(m.manifests.All())))
	return nil
}

// LatestManifest retrieves the active manifest.
func (m *Authority) LatestManifest() (*manifest.Manifest, error) {
	return m.manifests.Latest()
}

// createSeedEngine populates m.se.
//
// It is fine to call this function concurrently. After it returns, m.se is guaranteed to be
// non-nil.
func (m *Authority) createSeedEngine() error {
	// TODO(burgerdev): return this seed to the user
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
	return nil
}

type appendableList[T any] interface {
	Append(T)
	All() []T
	Latest() (T, error)
}
