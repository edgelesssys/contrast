// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package tdx

import (
	"context"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/attestation/reportdata"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-tdx-guest/proto/tdx"
	"github.com/google/go-tdx-guest/validate"
	"github.com/google/go-tdx-guest/verify"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Validator validates attestation statements.
type Validator struct {
	verifyOpts      *verify.Options
	validateOptsGen validateOptsGenerator
	reportSetter    attestation.ReportSetter
	logger          *slog.Logger
	name            string
}

type validateOptsGenerator interface {
	TDXValidateOpts(report *tdx.QuoteV4) (*validate.Options, error)
}

// StaticValidateOptsGenerator returns validate.Options generator that returns
// static validation options.
type StaticValidateOptsGenerator struct {
	Opts *validate.Options
}

// TDXValidateOpts return the TDX validation options.
func (v *StaticValidateOptsGenerator) TDXValidateOpts(_ *tdx.QuoteV4) (*validate.Options, error) {
	return v.Opts, nil
}

// NewValidator returns a new Validator.
func NewValidator(VerifyOpts *verify.Options, optsGen validateOptsGenerator, log *slog.Logger, name string) *Validator {
	return &Validator{
		verifyOpts:      VerifyOpts,
		validateOptsGen: optsGen,
		logger:          log,
		name:            name,
	}
}

// NewValidatorWithReportSetter returns a new Validator with a report setter.
func NewValidatorWithReportSetter(verifyOpts *verify.Options, optsGen validateOptsGenerator, log *slog.Logger, reportSetter attestation.ReportSetter, name string) *Validator {
	v := NewValidator(verifyOpts, optsGen, log, name)
	v.reportSetter = reportSetter
	return v
}

// OID returns the OID for the raw TDX report extension used by the validator.
func (v *Validator) OID() asn1.ObjectIdentifier {
	return oid.RawTDXReport
}

// Validate a TDX attestation.
func (v *Validator) Validate(ctx context.Context, attDocRaw []byte, nonce []byte, peerPublicKey []byte) (err error) {
	// TODO(freax13): Validate the memory integrity mode (logical vs cryptographic) in the provisioning certificate.
	//                https://github.com/google/go-tdx-guest/pull/51

	v.logger.Info("Validate called", "name", v.name, "nonce", hex.EncodeToString(nonce))
	defer func() {
		if err != nil {
			v.logger.Debug("Validate failed", "name", v.name, "nonce", hex.EncodeToString(nonce), "error", err)
		} else {
			v.logger.Info("Validate succeeded", "name", v.name, "nonce", hex.EncodeToString(nonce))
		}
	}()

	// Parse the attestation document.

	quote := &tdx.QuoteV4{}
	if err := proto.Unmarshal(attDocRaw, quote); err != nil {
		return fmt.Errorf("unmarshaling attestation: %w", err)
	}
	v.logger.Info("Quote decoded", "quote", protojson.MarshalOptions{Multiline: false}.Format(quote))

	// Verify the report signature.

	if err := verify.TdxQuoteContext(ctx, quote, v.verifyOpts); err != nil {
		return fmt.Errorf("verifying report signature: %w", err)
	}
	v.logger.Info("Successfully verified report signature")

	// Build the validation options.

	reportDataExpected := reportdata.Construct(peerPublicKey, nonce)
	validateOpts, err := v.validateOptsGen.TDXValidateOpts(quote)
	if err != nil {
		return fmt.Errorf("generating validation options: %w", err)
	}
	validateOpts.TdQuoteBodyOptions.ReportData = reportDataExpected[:]

	// Validate the report data.

	if err := validate.TdxQuote(quote, validateOpts); err != nil {
		return fmt.Errorf("validating report data: %w", err)
	}

	if v.reportSetter != nil {
		report := tdxReport{quote: quote}
		v.reportSetter.SetReport(report)
	}
	return nil
}

// String returns the name as identifier of the validator.
func (v *Validator) String() string {
	return v.name
}

type tdxReport struct {
	quote *tdx.QuoteV4
}

func (t tdxReport) HostData() []byte {
	return t.quote.TdQuoteBody.MrConfigId[:32]
}

func (t tdxReport) ClaimsToCertExtension() ([]pkix.Extension, error) {
	return claimsToCertExtension(t.quote)
}
