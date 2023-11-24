package main

import (
	"fmt"
	"sync"

	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
	"github.com/katexochen/coordinator-kbs/internal/ca"
)

type meshAuthority struct {
	ca       *ca.CA
	certs    map[string][]byte
	certsMux sync.RWMutex
	manifest *Manifest
}

func newMeshAuthority(manifest *Manifest) (*meshAuthority, error) {
	caInstance, err := ca.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create CA: %w", err)
	}
	return &meshAuthority{
		ca:       caInstance,
		certs:    make(map[string][]byte),
		manifest: manifest,
	}, nil
}

func (m *meshAuthority) SNPValidateOpts(report *sevsnp.Report) (*validate.Options, error) {
	hostData := NewHexString(report.HostData)
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
