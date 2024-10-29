// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package atlsinsecure

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/oid"
)

const MagicHostData = "insecure"

// Validator validates attestation statements.
type Validator struct {
	reportSetter attestation.ReportSetter
	logger       *slog.Logger
}

// NewValidator returns a new Validator.
func NewValidator(log *slog.Logger) *Validator {
	return &Validator{
		logger: log,
	}
}

// NewValidatorWithReportSetter returns a new Validator with a report setter.
func NewValidatorWithReportSetter(log *slog.Logger, reportSetter attestation.ReportSetter) *Validator {
	v := NewValidator(log)
	v.reportSetter = reportSetter
	return v
}

// OID returns the OID of the validator.
func (v *Validator) OID() asn1.ObjectIdentifier {
	return oid.RawInsecureReport
}

// Validate a TDX attestation.
func (v *Validator) Validate(_ []byte, nonce []byte, _ []byte) error {
	v.logger.Info("Validate called", "nonce", hex.EncodeToString(nonce))

	// Parse the attestation document.

	if v.reportSetter != nil {
		// We don't know what the policy hash was that this podVM was started with,
		// since it is not passed to SNP/TDX when the VM is started.
		report := insecureReport{hostData: []byte(MagicHostData)}
		v.reportSetter.SetReport(report)
	}

	v.logger.Info("Validate finished successfully")
	return nil
}

type insecureReport struct {
	hostData []byte
}

func (i insecureReport) HostData() []byte {
	return i.hostData
}

func (i insecureReport) ClaimsToCertExtension() ([]pkix.Extension, error) {
	return []pkix.Extension{}, nil
}
