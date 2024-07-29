// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package snp

import (
	"context"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"slices"

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
	callbackers            []validateCallbacker
	logger                 *slog.Logger
}

type validateCallbacker interface {
	ValidateCallback(ctx context.Context, report *sevsnp.Report, validatorOID asn1.ObjectIdentifier,
		reportRaw, nonce, peerPublicKey []byte) error
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

// NewValidatorWithCallbacks returns a new Validator with callbacks.
func NewValidatorWithCallbacks(VerifyOpts *verify.Options, ValidateOpts *validate.Options, allowedHostDataEntries []manifest.HexString,
	log *slog.Logger, callbacks ...validateCallbacker,
) *Validator {
	return &Validator{
		verifyOpts:             VerifyOpts,
		validateOpts:           ValidateOpts,
		allowedHostDataEntries: allowedHostDataEntries,
		callbackers:            callbacks,
		logger:                 log,
	}
}

// OID returns the OID of the validator.
func (v *Validator) OID() asn1.ObjectIdentifier {
	return oid.RawSNPReport
}

// Validate a TPM based attestation.
func (v *Validator) Validate(ctx context.Context, attDocRaw []byte, nonce []byte, peerPublicKey []byte) (err error) {
	v.logger.Info("Validate called", "nonce", hex.EncodeToString(nonce))

	// Parse the attestation document.

	attestation := &sevsnp.Attestation{}
	if err := proto.Unmarshal(attDocRaw, attestation); err != nil {
		return fmt.Errorf("unmarshaling attestation: %w", err)
	}

	if attestation.Report == nil {
		return fmt.Errorf("attestation missing report")
	}
	reportRaw, err := abi.ReportToAbiBytes(attestation.Report)
	if err != nil {
		return fmt.Errorf("converting report to abi format: %w", err)
	}
	v.logger.Info("Report decoded", "reportRaw", hex.EncodeToString(reportRaw))

	// Report signature verification.

	if err := verify.SnpAttestation(attestation, v.verifyOpts); err != nil {
		return fmt.Errorf("verifying report: %w", err)
	}
	v.logger.Info("Successfully verified report signature")

	// Build the validation options.

	reportDataExpected := reportdata.Construct(peerPublicKey, nonce)
	v.validateOpts.ReportData = reportDataExpected[:]
	if err := validate.SnpAttestation(attestation, v.validateOpts); err != nil {
		return fmt.Errorf("validating report claims: %w", err)
	}
	v.logger.Info("Successfully validated report data")

	// Validate the host data.

	if !slices.ContainsFunc(v.allowedHostDataEntries, func(entry manifest.HexString) bool {
		return manifest.NewHexString(attestation.Report.HostData) == entry
	}) {
		return fmt.Errorf("host data not allowed (found: %v allowed: %v)", attestation.Report.HostData, v.allowedHostDataEntries)
	}

	// Run callbacks.

	for _, callbacker := range v.callbackers {
		if err := callbacker.ValidateCallback(
			ctx, attestation.Report, v.OID(), reportRaw, nonce, peerPublicKey,
		); err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	v.logger.Info("Validate finished successfully")
	return nil
}
