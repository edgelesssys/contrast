// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package atls

import (
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/klauspost/cpuid/v2"
)

// PlatformIssuer creates an attestation issuer for the current platform.
func PlatformIssuer(log *slog.Logger) (Issuer, error) {
	cpuid.Detect()
	switch {
	case cpuid.CPU.Supports(cpuid.SEV_SNP):
		return snp.NewIssuer(
			logger.NewWithAttrs(logger.NewNamed(log, "issuer"), map[string]string{"tee-type": "snp"}),
		), nil
	case cpuid.CPU.Supports(cpuid.TDX_GUEST):
		return tdx.NewIssuer(
			logger.NewWithAttrs(logger.NewNamed(log, "issuer"), map[string]string{"tee-type": "tdx"}),
		), nil
	default:
		return nil, fmt.Errorf("unsupported platform: %T", cpuid.CPU)
	}
}
