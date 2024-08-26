// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package snp

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"slices"

	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/attestation/reportdata"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
	"github.com/google/go-sev-guest/verify"
	"google.golang.org/protobuf/proto"
)

// Validator validates attestation statements.
type Validator struct {
	verifyOpts             *verify.Options
	validateOpts           *validate.Options
	allowedHostDataEntries []manifest.HexString // Allowed host data entries in the report. If any of these is present, the report is considered valid.
	reportSetter           attestation.ReportSetter
	logger                 *slog.Logger
}

// NewValidator returns a new Validator.
func NewValidator(VerifyOpts *verify.Options, ValidateOpts *validate.Options, allowedHostDataEntries []manifest.HexString, log *slog.Logger) *Validator {
	return &Validator{
		verifyOpts:             VerifyOpts,
		validateOpts:           ValidateOpts,
		allowedHostDataEntries: allowedHostDataEntries,
		logger:                 log,
	}
}

// NewValidatorWithReportSetter returns a new Validator with a report setter.
func NewValidatorWithReportSetter(VerifyOpts *verify.Options, ValidateOpts *validate.Options, allowedHostDataEntries []manifest.HexString,
	log *slog.Logger, reportSetter attestation.ReportSetter,
) *Validator {
	return &Validator{
		verifyOpts:             VerifyOpts,
		validateOpts:           ValidateOpts,
		allowedHostDataEntries: allowedHostDataEntries,
		reportSetter:           reportSetter,
		logger:                 log,
	}
}

// OID returns the OID of the validator.
func (v *Validator) OID() asn1.ObjectIdentifier {
	return oid.RawSNPReport
}

// Validate a TPM based attestation.
func (v *Validator) Validate(attDocRaw []byte, nonce []byte, peerPublicKey []byte) (err error) {
	v.logger.Info("Validate called", "nonce", hex.EncodeToString(nonce))

	// Parse the attestation document.

	attestationData := &sevsnp.Attestation{}
	if err := proto.Unmarshal(attDocRaw, attestationData); err != nil {
		return fmt.Errorf("unmarshaling attestation: %w", err)
	}

	if attestationData.Report == nil {
		return fmt.Errorf("attestation missing report")
	}
	reportRaw, err := abi.ReportToAbiBytes(attestationData.Report)
	if err != nil {
		return fmt.Errorf("converting report to abi format: %w", err)
	}
	v.logger.Info("Report decoded", "reportRaw", hex.EncodeToString(reportRaw))

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
	v.logger.Info("Successfully validated report data")

	// Validate the host data.

	if !slices.ContainsFunc(v.allowedHostDataEntries, func(entry manifest.HexString) bool {
		return manifest.NewHexString(attestationData.Report.HostData) == entry
	}) {
		return fmt.Errorf("host data not allowed (found: %v allowed: %v)", attestationData.Report.HostData, v.allowedHostDataEntries)
	}

	if v.reportSetter != nil {
		report := snpReport{report: attestationData.Report}
		v.reportSetter.SetReport(report)
	}

	v.logger.Info("Validate finished successfully")
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
