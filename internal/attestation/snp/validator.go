// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package snp

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/attestation/reportdata"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-sev-guest/kds"
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
	name         string
}

// NewValidator returns a new Validator.
func NewValidator(VerifyOpts *verify.Options, ValidateOpts *validate.Options, log *slog.Logger, name string) *Validator {
	return &Validator{
		verifyOpts:   VerifyOpts,
		validateOpts: ValidateOpts,
		logger:       log,
		name:         name,
	}
}

// NewValidatorWithReportSetter returns a new Validator with a report setter.
func NewValidatorWithReportSetter(VerifyOpts *verify.Options, ValidateOpts *validate.Options,
	log *slog.Logger, reportSetter attestation.ReportSetter, name string,
) *Validator {
	return &Validator{
		verifyOpts:   VerifyOpts,
		validateOpts: ValidateOpts,
		reportSetter: reportSetter,
		logger:       log,
		name:         name,
	}
}

// OID returns the OID for the raw SNP report extension used by the validator.
func (v *Validator) OID() asn1.ObjectIdentifier {
	return oid.RawSNPReport
}

// Validate a TPM based attestation.
func (v *Validator) Validate(attDocRaw []byte, nonce []byte, peerPublicKey []byte) (err error) {
	v.logger.Info("Validate called", "name", v.name, "nonce", hex.EncodeToString(nonce))
	defer func() {
		if err != nil {
			v.logger.Debug("Validate failed", "name", v.name, "nonce", hex.EncodeToString(nonce), "error", err)
		} else {
			v.logger.Info("Validate succeeded", "name", v.name, "nonce", hex.EncodeToString(nonce))
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

	// CRL validity and expiration is checked as part of verify.SnpAttestation.
	if err := addCRLtoVerifyOptions(attestationData, v.verifyOpts); err != nil {
		// Log error but continue, the client can still request the CRL/VCEK from the KDS.
		v.logger.Info("could not use cached CRL from Coordinator aTLS handshake", slog.String("error", err.Error()))
	}

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

// String returns the name as identifier of the validator.
func (v *Validator) String() string {
	return v.name
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

// addCRLtoVerifyOptions adds the CRL from the attestation data to the verify options.
// The peer stores the CRL in the certificate chain extras so we don't need to request it from the KDS.
func addCRLtoVerifyOptions(attestationData *sevsnp.Attestation, verifyOpts *verify.Options) error {
	if verifyOpts.TrustedRoots == nil {
		return errors.New("no trusted roots found in verify options")
	}
	if attestationData.CertificateChain == nil {
		return errors.New("no certificate chain found in attestation data")
	}
	if attestationData.CertificateChain.Extras == nil {
		return errors.New("no extras found in certificate chain of attestation data")
	}

	crlRaw, ok := attestationData.CertificateChain.Extras[constants.SNPCertChainExtrasCRLKey]
	if !ok {
		return errors.New("no CRL found in attestation data")
	}
	crl, err := x509.ParseRevocationList(crlRaw)
	if err != nil {
		return errors.New("could not parse CRL from attestation data")
	}

	productLine := kds.ProductLine(attestationData.GetProduct())

	for _, tr := range verifyOpts.TrustedRoots[productLine] {
		tr.CRL = crl
	}

	return nil
}
