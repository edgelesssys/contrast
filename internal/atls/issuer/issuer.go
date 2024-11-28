// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package issuer

import (
	"context"
	"encoding/asn1"
	"fmt"
	"log/slog"

	snpissuer "github.com/edgelesssys/contrast/internal/attestation/snp/issuer"
	tdxissuer "github.com/edgelesssys/contrast/internal/attestation/tdx/issuer"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/klauspost/cpuid/v2"
)

// New creates an attestation issuer for the current platform.
func New(log *slog.Logger) (Issuer, error) {
	cpuid.Detect()
	switch {
	case cpuid.CPU.Supports(cpuid.SEV_SNP):
		return snpissuer.New(
			logger.NewWithAttrs(logger.NewNamed(log, "issuer"), map[string]string{"tee-type": "snp"}),
		), nil
	case cpuid.CPU.Supports(cpuid.TDX_GUEST):
		return tdxissuer.New(
			logger.NewWithAttrs(logger.NewNamed(log, "issuer"), map[string]string{"tee-type": "tdx"}),
		), nil
	default:
		return nil, fmt.Errorf("unsupported platform: %T", cpuid.CPU)
	}
}

// Issuer issues an attestation document.
type Issuer interface {
	Getter
	Issue(ctx context.Context, userData []byte, nonce []byte) (quote []byte, err error)
}

// Getter returns an ASN.1 Object Identifier.
type Getter interface {
	OID() asn1.ObjectIdentifier
}
