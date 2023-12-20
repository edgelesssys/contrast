/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: AGPL-3.0-only
*/

package snp

import (
	"context"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
	"github.com/google/go-sev-guest/verify"
)

type Validator struct {
	validateOptsGen validateOptsGenerator
	callbackers     []validateCallbacker
	logger          *slog.Logger
}

type validateCallbacker interface {
	ValidateCallback(ctx context.Context, report *sevsnp.Report, nonce []byte, peerPublicKey []byte) error
}

type validateOptsGenerator interface {
	SNPValidateOpts(report *sevsnp.Report) (*validate.Options, error)
}

type StaticValidateOptsGenerator struct {
	Opts *validate.Options
}

func (v *StaticValidateOptsGenerator) SNPValidateOpts(report *sevsnp.Report) (*validate.Options, error) {
	return v.Opts, nil
}

func NewValidator(optsGen validateOptsGenerator, log *slog.Logger) *Validator {
	return &Validator{
		validateOptsGen: optsGen,
		logger:          log.WithGroup("snp-validator"),
	}
}

func NewValidatorWithCallbacks(optsGen validateOptsGenerator, log *slog.Logger, callbacks ...validateCallbacker) *Validator {
	return &Validator{
		validateOptsGen: optsGen,
		callbackers:     callbacks,
		logger:          log.WithGroup("snp-validator"),
	}
}

func (v *Validator) OID() asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{1, 3, 9900, 77, 77}
}

// Validate a TPM based attestation.
func (v *Validator) Validate(ctx context.Context, attDocRaw []byte, nonce []byte, peerPublicKey []byte) (err error) {
	v.logger.Info("Validate called", "nonce", hex.EncodeToString(nonce))
	defer func() {
		if err != nil {
			v.logger.Error("Failed to validate attestation document: %s", err)
		}
	}()

	// Parse the attestation document.

	reportRaw := make([]byte, base64.StdEncoding.DecodedLen(len(attDocRaw)))
	if _, err = base64.StdEncoding.Decode(reportRaw, attDocRaw); err != nil {
		return err
	}
	v.logger.Info("Report decoded", "reportRaw", hex.EncodeToString(reportRaw))

	report, err := abi.ReportToProto(reportRaw)
	if err != nil {
		return fmt.Errorf("converting report to proto: %w", err)
	}

	// Report signature verification.

	verifyOpts := &verify.Options{}
	attestation, err := verify.GetAttestationFromReport(report, verifyOpts)
	if err != nil {
		return fmt.Errorf("getting attestation from report: %w", err)
	}
	if err := verify.SnpAttestation(attestation, verifyOpts); err != nil {
		return fmt.Errorf("verifying report: %w", err)
	}
	v.logger.Info("Successfully verified report signature")

	// Validate the report data.

	reportDataExpected := constructReportData(peerPublicKey, nonce)
	validateOpts, err := v.validateOptsGen.SNPValidateOpts(report)
	if err != nil {
		return fmt.Errorf("generating validation options: %w", err)
	}
	validateOpts.ReportData = reportDataExpected[:]
	if err := validate.SnpAttestation(attestation, validateOpts); err != nil {
		return fmt.Errorf("validating report claims: %w", err)
	}
	v.logger.Info("Successfully validated report data")

	// Run callbacks.

	for _, callbacker := range v.callbackers {
		if err := callbacker.ValidateCallback(ctx, report, nonce, peerPublicKey); err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	v.logger.Info("Validate finished successfully")
	return nil
}
