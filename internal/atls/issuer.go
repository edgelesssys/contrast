// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package atls

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/klauspost/cpuid/v2"
)

// PlatformIssuer creates an attestation issuer for the current platform.
func PlatformIssuer(log *slog.Logger) (Issuer, error) {
	var issuer Issuer
	switch {
	case cpuid.CPU.Supports(cpuid.SEV_SNP):
		issuer = snp.NewIssuer(
			logger.NewWithAttrs(logger.NewNamed(log, "issuer"), map[string]string{"tee-type": "snp"}),
		)
	case cpuid.CPU.Supports(cpuid.TDX_GUEST):
		issuer = tdx.NewIssuer(
			logger.NewWithAttrs(logger.NewNamed(log, "issuer"), map[string]string{"tee-type": "tdx"}),
		)
	default:
		return nil, fmt.Errorf("unsupported platform: %T", cpuid.CPU)
	}

	if hasTPM() {
		issuer = &vtpmIssuer{Issuer: issuer}
	}
	return issuer, nil
}

var tpmDevice = "/dev/tpm0"

func hasTPM() bool {
	f, err := os.Open(tpmDevice)
	if err == nil {
		f.Close()
		return true
	}
	// If the device does not exist, we don't have a TPM.
	// If the device exists but there is no backing TPM, the Open call fails with ENODEV.
	return false
}

// vtpmIssuer issues attestation statements for VMs with a TPM.
// TODO(burgerdev): this is currently a mock that just delegates to an underlying issuer.
type vtpmIssuer struct {
	Issuer
}
