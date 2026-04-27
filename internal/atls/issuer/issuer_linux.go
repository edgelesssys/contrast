// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package issuer

import (
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/insecure"
	snpissuer "github.com/edgelesssys/contrast/internal/attestation/snp/issuer"
	tdxissuer "github.com/edgelesssys/contrast/internal/attestation/tdx/issuer"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/klauspost/cpuid/v2"
)

// New creates an attestation issuer for the current platform.
func New(log *slog.Logger, collateralProxy string) (atls.Issuer, error) {
	cpuid.Detect()
	switch {
	case cpuid.CPU.Supports(cpuid.SEV_SNP):
		return snpissuer.New(
			logger.NewWithAttrs(logger.NewNamed(log, "issuer"), map[string]string{"tee-type": "snp"}),
			collateralProxy,
		), nil
	case cpuid.CPU.Supports(cpuid.TDX_GUEST):
		return tdxissuer.New(
			logger.NewWithAttrs(logger.NewNamed(log, "issuer"), map[string]string{"tee-type": "tdx"}),
		), nil
	default:
		allowed, err := insecure.AttestationAllowed()
		if err != nil {
			return nil, fmt.Errorf("checking insecure attestation opt-in: %w", err)
		}
		if !allowed {
			return nil, fmt.Errorf("unsupported platform: %T", cpuid.CPU)
		}
		log.Warn("No TEE platform detected, using insecure attestation issuer")
		return insecure.NewIssuer(), nil
	}
}
