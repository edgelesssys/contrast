package main

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"log"
	"sync"

	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
	"github.com/katexochen/coordinator-kbs/internal/ca"
	"github.com/katexochen/coordinator-kbs/internal/manifest"
)

type meshAuthority struct {
	ca       *ca.CA
	certs    map[string][]byte
	certsMux sync.RWMutex
	manifest *manifest.Manifest
}

func newMeshAuthority(ca *ca.CA, manifest *manifest.Manifest) (*meshAuthority, error) {
	return &meshAuthority{
		ca:       ca,
		certs:    make(map[string][]byte),
		manifest: manifest,
	}, nil
}

func (m *meshAuthority) SNPValidateOpts(report *sevsnp.Report) (*validate.Options, error) {
	hostData := manifest.NewHexString(report.HostData)
	if _, ok := m.manifest.Policies[hostData]; !ok {
		return nil, fmt.Errorf("hostdata %s not found in manifest", hostData)
	}

	trustedIDKeyDigestHashes, err := m.manifest.ReferenceValues.SNP.TrustedIDKeyHashes.ByteSlices()
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
			BlSpl:    m.manifest.ReferenceValues.SNP.MinimumTCB.BootloaderVersion.UInt8(),
			TeeSpl:   m.manifest.ReferenceValues.SNP.MinimumTCB.TEEVersion.UInt8(),
			SnpSpl:   m.manifest.ReferenceValues.SNP.MinimumTCB.SNPVersion.UInt8(),
			UcodeSpl: m.manifest.ReferenceValues.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
		},
		MinimumLaunchTCB: kds.TCBParts{
			BlSpl:    m.manifest.ReferenceValues.SNP.MinimumTCB.BootloaderVersion.UInt8(),
			TeeSpl:   m.manifest.ReferenceValues.SNP.MinimumTCB.TEEVersion.UInt8(),
			SnpSpl:   m.manifest.ReferenceValues.SNP.MinimumTCB.SNPVersion.UInt8(),
			UcodeSpl: m.manifest.ReferenceValues.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
		},
		PermitProvisionalFirmware: true,
		TrustedIDKeyHashes:        trustedIDKeyDigestHashes,
		RequireIDBlock:            true,
	}, nil
}

func (m *meshAuthority) ValidateCallback(ctx context.Context, report *sevsnp.Report, nonce []byte, peerPubKeyBytes []byte) error {
	hostData := manifest.NewHexString(report.HostData)
	commonName, ok := m.manifest.Policies[hostData]
	if !ok {
		return fmt.Errorf("report data %s not found in manifest", hostData)
	}

	peerPubKey, err := x509.ParsePKIXPublicKey(peerPubKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse peer public key: %w", err)
	}

	var extensions []pkix.Extension // TODO
	cert, err := m.ca.NewAttestedMeshCert(commonName, extensions, peerPubKey)
	if err != nil {
		return fmt.Errorf("failed to issue new attested mesh cert: %w", err)
	}

	peerPubKeyHash := sha256.Sum256(peerPubKeyBytes)
	peerPublicKeyHashStr := hex.EncodeToString(peerPubKeyHash[:])
	log.Printf("peerPublicKeyHashStr: %v", peerPublicKeyHashStr)

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
