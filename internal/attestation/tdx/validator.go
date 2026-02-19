// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package tdx

import (
	"bytes"
	"context"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"slices"

	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/attestation/tdx/quote"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-tdx-guest/pcs"
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
	allowedPIIDs    [][]byte
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
func NewValidator(VerifyOpts *verify.Options, optsGen validateOptsGenerator, allowedPIIDs [][]byte, log *slog.Logger, name string) *Validator {
	return &Validator{
		verifyOpts:      VerifyOpts,
		validateOptsGen: optsGen,
		allowedPIIDs:    allowedPIIDs,
		logger:          log,
		name:            name,
	}
}

// NewValidatorWithReportSetter returns a new Validator with a report setter.
func NewValidatorWithReportSetter(verifyOpts *verify.Options, optsGen validateOptsGenerator, allowedPIIDs [][]byte, log *slog.Logger, reportSetter attestation.ReportSetter, name string) *Validator {
	v := NewValidator(verifyOpts, optsGen, allowedPIIDs, log, name)
	v.reportSetter = reportSetter
	return v
}

// OID returns the OID for the raw TDX report extension used by the validator.
func (v *Validator) OID() asn1.ObjectIdentifier {
	return oid.RawTDXReport
}

// Validate a TDX attestation.
func (v *Validator) Validate(ctx context.Context, attDocRaw []byte, reportData []byte) (err error) {
	v.logger.Info("Validate called", "name", v.name, "report-data", hex.EncodeToString(reportData))
	defer func() {
		if err != nil {
			v.logger.Debug("Validate failed", "name", v.name, "report-data", hex.EncodeToString(reportData), "error", err)
		} else {
			v.logger.Info("Validate succeeded", "name", v.name, "report-data", hex.EncodeToString(reportData))
		}
	}()

	// Parse the attestation document.

	quotev4 := &tdx.QuoteV4{}
	if err := proto.Unmarshal(attDocRaw, quotev4); err != nil {
		return fmt.Errorf("unmarshaling attestation: %w", err)
	}
	v.logger.Debug("Quote decoded", "quote", protojson.MarshalOptions{Multiline: false}.Format(quotev4))

	// Verify the report signature.

	if len(quotev4.ExtraBytes) > 0 {
		extensions, err := quote.GetExtensions(quotev4)
		if err == nil {
			v.logger.Debug("extracted collateral from cert", "collateral", extensions.Collateral)
			// TODO(burgerdev): pass collateral to verifier
		} else {
			v.logger.Warn("error getting collateral from cert", "error", err)
		}
	}

	if err := verify.TdxQuoteContext(ctx, quotev4, v.verifyOpts); err != nil {
		return fmt.Errorf("verifying report signature: %w", err)
	}
	v.logger.Info("Successfully verified report signature")

	// Build the validation options.

	validateOpts, err := v.validateOptsGen.TDXValidateOpts(quotev4)
	if err != nil {
		return fmt.Errorf("generating validation options: %w", err)
	}
	validateOpts.TdQuoteBodyOptions.ReportData = reportData

	// Validate the report data.

	if err := validate.TdxQuote(quotev4, validateOpts); err != nil {
		return fmt.Errorf("validating report data: %w", err)
	}

	//
	// Additional checks.
	//

	// Check for allowed PIIDs.
	if len(v.allowedPIIDs) != 0 {
		piid, err := getPIID(quotev4)
		if err != nil {
			return fmt.Errorf("reading PIID from quote: %w", err)
		}
		if !slices.ContainsFunc(v.allowedPIIDs, func(id []byte) bool {
			return bytes.Equal(id, piid)
		}) {
			return fmt.Errorf("PIID %x not in allowed PIIDs", piid)
		}
	}

	if v.reportSetter != nil {
		report := &Report{Quote: quotev4}
		v.reportSetter.SetReport(report)
	}
	return nil
}

// String returns the name as identifier of the validator.
func (v *Validator) String() string {
	return v.name
}

// Report wraps a TDX quote to implement attestation.Report.
type Report struct {
	Quote *tdx.QuoteV4
}

// HostData allows extracting MRCONFIGID for manifest validation.
func (t Report) HostData() []byte {
	return t.Quote.TdQuoteBody.MrConfigId[:32]
}

// ClaimsToCertExtension converts the TDX quote claims to an X.509 certificate extension.
func (t Report) ClaimsToCertExtension() ([]pkix.Extension, error) {
	return claimsToCertExtension(t.Quote)
}

// getPIID extracts the PIID from the PCK certificate inside a TDX quote.
func getPIID(quotev4 *tdx.QuoteV4) ([]byte, error) {
	pck, err := quote.GetPCKCertificate(quotev4)
	if err != nil {
		return nil, fmt.Errorf("extracting PCK certificate: %w", err)
	}
	extensions, err := pcs.PckCertificateExtensions(pck)
	if err != nil {
		return nil, fmt.Errorf("extracting PCK certificate extensions: %w", err)
	}

	piid, err := hex.DecodeString(extensions.PIID)
	if err != nil {
		return nil, fmt.Errorf("decoding PIID from extension: %w", err)
	}

	const piidLen = 16
	if len(piid) != piidLen {
		return nil, fmt.Errorf("unexpected PIID length of %d, want %d", len(piid), piidLen)
	}

	return piid, nil
}
