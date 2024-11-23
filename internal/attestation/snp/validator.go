// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package snp

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/attestation/reportdata"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
	"github.com/google/go-sev-guest/verify"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Validator validates attestation statements.
type Validator struct {
	verifyOpts   *verify.Options
	validateOpts *validate.Options
	reportSetter attestation.ReportSetter
	logger       *slog.Logger
}

// NewValidator returns a new Validator.
func NewValidator(VerifyOpts *verify.Options, ValidateOpts *validate.Options, log *slog.Logger) *Validator {
	return &Validator{
		verifyOpts:   VerifyOpts,
		validateOpts: ValidateOpts,
		logger:       log,
	}
}

// NewValidatorWithReportSetter returns a new Validator with a report setter.
func NewValidatorWithReportSetter(VerifyOpts *verify.Options, ValidateOpts *validate.Options,
	log *slog.Logger, reportSetter attestation.ReportSetter,
) *Validator {
	return &Validator{
		verifyOpts:   VerifyOpts,
		validateOpts: ValidateOpts,
		reportSetter: reportSetter,
		logger:       log,
	}
}

// OID returns the OID of the validator.
func (v *Validator) OID() asn1.ObjectIdentifier {
	return oid.RawSNPReport
}

// Validate a TPM based attestation.
func (v *Validator) Validate(attDocRaw []byte, nonce []byte, peerPublicKey []byte) (err error) {
	v.logger.Info("Validate called", "nonce", hex.EncodeToString(nonce))
	defer func() {
		if err != nil {
			v.logger.Error("Validation failed", "error", err)
		} else {
			v.logger.Info("Validation successful")
		}
	}()

	// Parse the attestation document.

	attestationData := &sevsnp.Attestation{}
	if err := proto.Unmarshal(attDocRaw, attestationData); err != nil {
		return fmt.Errorf("unmarshaling attestation: %w", err)
	}

	if attestationData.Report == nil {
		return fmt.Errorf("attestation missing report")
	}
	v.logger.Info("Report decoded", "report", protojson.MarshalOptions{Multiline: false}.Format(attestationData.Report))

	// Report signature verification.

	if err := verify.SnpAttestation(attestationData, v.verifyOpts); err != nil {
		return fmt.Errorf("verifying report: %w", err)
	}
	v.logger.Info("Successfully verified report signature")

	// Build the validation options.

	reportDataExpected := reportdata.Construct(peerPublicKey, nonce)
	v.validateOpts.ReportData = reportDataExpected[:]
	if err := validate.SnpAttestation(attestationData, v.validateOpts); err != nil {
		return fmt.Errorf("validating report claims: %w", err)
	}

	if v.reportSetter != nil {
		report := snpReport{report: attestationData.Report}
		v.reportSetter.SetReport(report)
	}
	return nil
}

type snpReport struct {
	report *sevsnp.Report
}

func (s snpReport) HostData() []byte {
	return s.report.HostData
}

func (s snpReport) ClaimsToCertExtension() ([]pkix.Extension, error) {
	return claimsToCertExtension(s.report)
}
