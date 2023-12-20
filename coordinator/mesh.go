package main

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"

	"github.com/edgelesssys/nunki/internal/appendable"
	"github.com/edgelesssys/nunki/internal/ca"
	"github.com/edgelesssys/nunki/internal/manifest"
	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
)

type meshAuthority struct {
	ca        *ca.CA
	certs     map[string][]byte
	certsMux  sync.RWMutex
	manifests appendableList[manifest.Manifest]
	logger    *slog.Logger
}

func newMeshAuthority(ca *ca.CA, log *slog.Logger) (*meshAuthority, error) {
	return &meshAuthority{
		ca:        ca,
		certs:     make(map[string][]byte),
		manifests: new(appendable.Appendable[manifest.Manifest]),
		logger:    log.WithGroup("mesh-authority"),
	}, nil
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
		RequireIDBlock:            true,
	}, nil
}

func (m *meshAuthority) ValidateCallback(ctx context.Context, report *sevsnp.Report, nonce []byte, peerPubKeyBytes []byte) error {
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

	var extensions []pkix.Extension // TODO
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

func (m *meshAuthority) GetManifest() *manifest.Manifest {
	mnfst, _ := m.manifests.Latest()
	return mnfst
}

func (m *meshAuthority) SetManifest(mnfst *manifest.Manifest) error {
	return m.manifests.Append(mnfst)
}

type appendableList[T any] interface {
	Append(*T) error
	All() []*T
	Latest() (*T, error)
}
