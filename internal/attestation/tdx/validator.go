// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package tdx

import (
	"context"
	"crypto/x509"
	_ "embed"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/attestation/reportdata"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-tdx-guest/abi"
	"github.com/google/go-tdx-guest/proto/tdx"
	"github.com/google/go-tdx-guest/validate"
	"github.com/google/go-tdx-guest/verify"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/proto"
)

// Even though the vendored file has "SGX" in its name, it is the general "Provisioning Certificate for ECDSA Attestation"
// from Intel and used for both SGX *and* TDX.
//
// See https://api.portal.trustedservices.intel.com/content/documentation.html#pcs for more information.
//
// File Source: https://certificates.trustedservices.intel.com/Intel_SGX_Provisioning_Certification_RootCA.pem
//
//go:embed Intel_SGX_Provisioning_Certification_RootCA.pem
var tdxRootCert []byte

// Validator validates attestation statements.
type Validator struct {
	validateOptsGen validateOptsGenerator
	callbackers     []validateCallbacker
	logger          *slog.Logger
	metrics         metrics
}

type metrics struct {
	attestationFailures prometheus.Counter
}

type validateCallbacker interface {
	ValidateCallback(ctx context.Context, quote *tdx.QuoteV4, validatorOID asn1.ObjectIdentifier,
		reportRaw, nonce, peerPublicKey []byte) error
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
func NewValidator(optsGen validateOptsGenerator, log *slog.Logger) *Validator {
	return &Validator{
		validateOptsGen: optsGen,
		logger:          log,
	}
}

// NewValidatorWithCallbacks returns a new Validator with callbacks.
func NewValidatorWithCallbacks(optsGen validateOptsGenerator, log *slog.Logger, attestationFailures prometheus.Counter, callbacks ...validateCallbacker) *Validator {
	v := NewValidator(optsGen, log)
	v.callbackers = callbacks
	v.metrics = metrics{attestationFailures: attestationFailures}
	return v
}

// OID returns the OID of the validator.
func (v *Validator) OID() asn1.ObjectIdentifier {
	return oid.RawTDXReport
}

// Validate a TDX attestation.
func (v *Validator) Validate(ctx context.Context, attDocRaw []byte, nonce []byte, peerPublicKey []byte) (err error) {
	// TODO(freax13): Validate the memory integrity mode (logical vs cryptographic) in the provisioning certificate.

	v.logger.Info("Validate called", "nonce", hex.EncodeToString(nonce))
	defer func() {
		if err != nil {
			v.logger.Error("Failed to validate attestation document", "err", err)
			if v.metrics.attestationFailures != nil {
				v.metrics.attestationFailures.Inc()
			}
		}
	}()

	// Parse the attestation document.

	quote := &tdx.QuoteV4{}
	if err := proto.Unmarshal(attDocRaw, quote); err != nil {
		return fmt.Errorf("unmarshaling attestation: %w", err)
	}

	quoteRaw, err := abi.QuoteToAbiBytes(quote)
	if err != nil {
		return fmt.Errorf("converting quote to abi format: %w", err)
	}
	v.logger.Info("Quote decoded", "quoteRaw", hex.EncodeToString(quoteRaw))

	// Build the verification options.

	verifyOpts := verify.DefaultOptions()
	rootCerts, err := trustedRoots()
	if err != nil {
		return fmt.Errorf("getting trusted roots: %w", err)
	}
	verifyOpts.TrustedRoots = rootCerts
	verifyOpts.CheckRevocations = true
	verifyOpts.GetCollateral = true
	// TODO(freax13): Set .Getter with a caching HTTP getter implementation.

	// Verify the report signature.

	if err := verify.TdxQuote(quote, verifyOpts); err != nil {
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
	v.logger.Info("Successfully validated report data")

	// Run callbacks.

	for _, callbacker := range v.callbackers {
		if err := callbacker.ValidateCallback(
			ctx, quote, v.OID(), quoteRaw, nonce, peerPublicKey,
		); err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	v.logger.Info("Validate finished successfully")
	return nil
}

func trustedRoots() (*x509.CertPool, error) {
	rootCerts := x509.NewCertPool()
	if ok := rootCerts.AppendCertsFromPEM(tdxRootCert); !ok {
		return nil, fmt.Errorf("failed to append root certificate")
	}
	return rootCerts, nil
}
