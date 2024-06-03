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

	"github.com/edgelesssys/contrast/internal/appendable"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
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
	ca         *ca.CA
	bundles    map[string]Bundle
	bundlesMux sync.RWMutex
	manifests  appendableList[*manifest.Manifest]
	logger     *slog.Logger
}

// New creates a new Authority instance.
func New(caInstance *ca.CA, log *slog.Logger) *Authority {
	return &Authority{
		ca:        caInstance,
		bundles:   make(map[string]Bundle),
		manifests: new(appendable.Appendable[*manifest.Manifest]),
		logger:    log.WithGroup("mesh-authority"),
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
	caInstance := m.ca

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
	return m.manifests.All(), m.ca
}

// SetManifest updates the active manifest.
func (m *Authority) SetManifest(mnfst *manifest.Manifest) error {
	if err := m.ca.RotateIntermCerts(); err != nil {
		return fmt.Errorf("rotating intermediate certificates: %w", err)
	}
	m.manifests.Append(mnfst)
	return nil
}

// LatestManifest retrieves the active manifest.
func (m *Authority) LatestManifest() (*manifest.Manifest, error) {
	return m.manifests.Latest()
}

type appendableList[T any] interface {
	Append(T)
	All() []T
	Latest() (T, error)
}
