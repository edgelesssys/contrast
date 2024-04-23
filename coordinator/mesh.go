// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

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

type meshAuthority struct {
	ca        *ca.CA
	certs     map[string][]byte
	certsMux  sync.RWMutex
	manifests appendableList[*manifest.Manifest]
	logger    *slog.Logger
}

func newMeshAuthority(ca *ca.CA, log *slog.Logger) *meshAuthority {
	return &meshAuthority{
		ca:        ca,
		certs:     make(map[string][]byte),
		manifests: new(appendable.Appendable[*manifest.Manifest]),
		logger:    log.WithGroup("mesh-authority"),
	}
}

func (m *meshAuthority) SNPValidateOpts(report *sevsnp.Report) (*validate.Options, error) {
	mnfst, err := m.manifests.Latest()
	if err != nil {
		return nil, fmt.Errorf("getting latest manifest: %w", err)
	}

	hostData := manifest.NewHexString(report.HostData)
	if _, ok := mnfst.Policies[hostData]; !ok {
		return nil, fmt.Errorf("hostdata %s not found in manifest", hostData)
	}

	trustedIDKeyDigestHashes, err := mnfst.ReferenceValues.SNP.TrustedIDKeyHashes.ByteSlices()
	if err != nil {
		return nil, fmt.Errorf("failed to convert TrustedIDKeyHashes from manifest to byte slices: %w", err)
	}

	return &validate.Options{
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
		TrustedIDKeyHashes:        trustedIDKeyDigestHashes,
		RequireIDBlock:            false, // TODO(malt3): re-enable once we control the full boot (including the id block)
	}, nil
}

func (m *meshAuthority) ValidateCallback(_ context.Context, report *sevsnp.Report,
	_ asn1.ObjectIdentifier, _, _, peerPubKeyBytes []byte,
) error {
	mnfst, err := m.manifests.Latest()
	if err != nil {
		return fmt.Errorf("getting latest manifest: %w", err)
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
	cert, err := m.ca.NewAttestedMeshCert(dnsNames, extensions, peerPubKey)
	if err != nil {
		return fmt.Errorf("failed to issue new attested mesh cert: %w", err)
	}

	peerPubKeyHash := sha256.Sum256(peerPubKeyBytes)
	peerPublicKeyHashStr := hex.EncodeToString(peerPubKeyHash[:])
	m.logger.Info("Validated peer", "peerPublicKeyHashStr", peerPublicKeyHashStr)

	m.certsMux.Lock()
	defer m.certsMux.Unlock()
	m.certs[peerPublicKeyHashStr] = cert

	return nil
}

func (m *meshAuthority) GetCert(peerPublicKeyHashStr string) ([]byte, error) {
	m.certsMux.RLock()
	defer m.certsMux.RUnlock()

	cert, ok := m.certs[peerPublicKeyHashStr]
	if !ok {
		return nil, fmt.Errorf("cert for peer public key %s not found", peerPublicKeyHashStr)
	}

	return cert, nil
}

func (m *meshAuthority) GetManifests() []*manifest.Manifest {
	return m.manifests.All()
}

func (m *meshAuthority) SetManifest(mnfst *manifest.Manifest) error {
	if err := m.ca.RotateIntermCerts(); err != nil {
		return fmt.Errorf("rotating intermediate certificates: %w", err)
	}
	m.manifests.Append(mnfst)
	return nil
}

func (m *meshAuthority) LatestManifest() (*manifest.Manifest, error) {
	return m.manifests.Latest()
}

type appendableList[T any] interface {
	Append(T)
	All() []T
	Latest() (T, error)
}
