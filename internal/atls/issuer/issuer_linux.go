// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package issuer

import (
	"log/slog"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/insecure"
	snpissuer "github.com/edgelesssys/contrast/internal/attestation/snp/issuer"
	tdxissuer "github.com/edgelesssys/contrast/internal/attestation/tdx/issuer"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/klauspost/cpuid/v2"
)

// New creates an attestation issuer for the current platform.
func New(log *slog.Logger) (atls.Issuer, error) {
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
		log.Warn("No TEE platform detected, using insecure attestation issuer")
		return insecure.NewIssuer(), nil
	}
}
