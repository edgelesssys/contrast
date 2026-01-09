// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package tdx

import (
	"bytes"
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log/slog"
	"slices"

	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-tdx-guest/proto/tdx"
	"github.com/google/go-tdx-guest/validate"
	"github.com/google/go-tdx-guest/verify"
	"golang.org/x/crypto/cryptobyte"
	cryptobyte_asn1 "golang.org/x/crypto/cryptobyte/asn1"
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
	// TODO(freax13): Validate the memory integrity mode (logical vs cryptographic) in the provisioning certificate.
	//                https://github.com/google/go-tdx-guest/pull/51

	v.logger.Info("Validate called", "name", v.name, "report-data", hex.EncodeToString(reportData))
	defer func() {
		if err != nil {
			v.logger.Debug("Validate failed", "name", v.name, "report-data", hex.EncodeToString(reportData), "error", err)
		} else {
			v.logger.Info("Validate succeeded", "name", v.name, "report-data", hex.EncodeToString(reportData))
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

	validateOpts, err := v.validateOptsGen.TDXValidateOpts(quote)
	if err != nil {
		return fmt.Errorf("generating validation options: %w", err)
	}
	validateOpts.TdQuoteBodyOptions.ReportData = reportData

	// Validate the report data.

	if err := validate.TdxQuote(quote, validateOpts); err != nil {
		return fmt.Errorf("validating report data: %w", err)
	}

	//
	// Additional checks.
	//

	// Check for allowed PIIDs.
	if len(v.allowedPIIDs) != 0 {
		piid, err := getPIID(quote)
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

// getPIID extracts the PIID from the PCK certificate inside a TDX quote.
func getPIID(quote *tdx.QuoteV4) ([]byte, error) {
	pckCertChain := quote.GetSignedData().GetCertificationData().GetQeReportCertificationData().GetPckCertificateChainData().PckCertChain

	// The certChain input is a concatenated list of PEM-encoded X.509 certificates.
	// https://download.01.org/intel-sgx/latest/dcap-latest/linux/docs/Intel_TDX_DCAP_Quoting_Library_API.pdf, A.3.9

	var pckBlock *pem.Block
	var pck *x509.Certificate
	for len(pckCertChain) > 0 {
		pckBlock, pckCertChain = pem.Decode(pckCertChain)
		if pckBlock == nil {
			break
		}
		candidate, err := x509.ParseCertificate(pckBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing PCK certificate: %w", err)
		}

		// PCK certificates have specified static names.
		// https://api.trustedservices.intel.com/documents/Intel_SGX_PCK_Certificate_CRL_Spec-1.5.pdf, 1.3.5
		if candidate.Subject.CommonName == "Intel SGX PCK Certificate" {
			pck = candidate
			break
		}
	}
	if pck == nil {
		return nil, fmt.Errorf("no PCK certificate found in TDX quote")
	}

	// The PCK certificate contains an SGX Extension, which is a nested list of ASN.1 objects.
	// https://api.trustedservices.intel.com/documents/Intel_SGX_PCK_Certificate_CRL_Spec-1.5.pdf, 1.3.5

	// Ideally, we would just be using
	// https://pkg.go.dev/github.com/google/go-tdx-guest/pcs#PckCertificateExtensions to access
	// these extensions, but it currently lacks the PIID field.
	// TODO(burgerdev): implement upstream

	var sgxExtensions cryptobyte.String
	for _, ext := range pck.Extensions {
		if !ext.Id.Equal(oid.SGXExtensionsOID) {
			continue
		}
		extValue := cryptobyte.String(ext.Value)
		if !extValue.ReadASN1(&sgxExtensions, cryptobyte_asn1.SEQUENCE) {
			return nil, fmt.Errorf("could not read SGX extensions from PCK cert")
		}
	}
	if sgxExtensions == nil {
		return nil, fmt.Errorf("no SGX extensions found on PCK certificate")
	}
	for !sgxExtensions.Empty() {
		var extension cryptobyte.String
		if !sgxExtensions.ReadASN1(&extension, cryptobyte_asn1.SEQUENCE) {
			return nil, fmt.Errorf("could not parse SGX extension")
		}
		var id asn1.ObjectIdentifier
		if !extension.ReadASN1ObjectIdentifier(&id) {
			return nil, fmt.Errorf("could not parse SGX extension OID")
		}
		if !id.Equal(oid.PlatformInstanceIDOID) {
			continue
		}
		var piid cryptobyte.String
		if !extension.ReadASN1(&piid, cryptobyte_asn1.OCTET_STRING) {
			return nil, fmt.Errorf("could not parse SGX extension value")
		}

		if len(piid) != 16 {
			return nil, fmt.Errorf("expected PIID of size 16, got %d", len(piid))
		}
		return piid, nil
	}
	return nil, fmt.Errorf("no PIID extension found in PCK SGX extensions")
}
