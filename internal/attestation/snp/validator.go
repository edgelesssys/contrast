// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package snp

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/idblock"
	"github.com/edgelesssys/contrast/internal/oid"
	snpmeasure "github.com/edgelesssys/contrast/internal/snp"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
	"github.com/google/go-sev-guest/verify"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Validator validates attestation statements.
type Validator struct {
	verifyOpts     *verify.Options
	validateOpts   *validate.Options
	allowedChipIDs [][]byte
	reportSetter   attestation.ReportSetter
	logger         *slog.Logger
	name           string
}

// NewValidator returns a new Validator.
func NewValidator(
	verifyOpts *verify.Options, validateOpts *validate.Options, allowedChipIDs [][]byte,
	log *slog.Logger, name string,
) *Validator {
	return &Validator{
		verifyOpts:     verifyOpts,
		validateOpts:   validateOpts,
		allowedChipIDs: allowedChipIDs,
		logger:         log,
		name:           name,
	}
}

// NewValidatorWithReportSetter returns a new Validator with a report setter.
func NewValidatorWithReportSetter(
	verifyOpts *verify.Options, validateOpts *validate.Options, allowedChipIDs [][]byte,
	log *slog.Logger, reportSetter attestation.ReportSetter, name string,
) *Validator {
	return &Validator{
		verifyOpts:     verifyOpts,
		validateOpts:   validateOpts,
		allowedChipIDs: allowedChipIDs,
		reportSetter:   reportSetter,
		logger:         log,
		name:           name,
	}
}

// OID returns the OID for the raw SNP report extension used by the validator.
func (v *Validator) OID() asn1.ObjectIdentifier {
	return oid.RawSNPReport
}

// Validate a SNP based attestation.
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

	attestationData := &sevsnp.Attestation{}
	if err := proto.Unmarshal(attDocRaw, attestationData); err != nil {
		return fmt.Errorf("unmarshaling attestation: %w", err)
	}

	if attestationData.Report == nil {
		return fmt.Errorf("attestation missing report")
	}
	v.logger.Debug("Report decoded", "report", protojson.MarshalOptions{Multiline: false}.Format(attestationData.Report))

	//
	//	Checkout dev-docs/kds.md for overview over VCEK/CRL retrieval/caching.
	//

	// CRL validity and expiration is checked as part of verify.SnpAttestation.
	if err := addCRLtoVerifyOptions(attestationData, v.verifyOpts); err != nil {
		// Log error but continue, the client can still request the CRL/VCEK from the KDS.
		v.logger.Info("could not use cached CRL from Coordinator aTLS handshake", slog.String("error", err.Error()))
	}

	// Report signature verification.
	if err := verify.SnpAttestationContext(ctx, attestationData, v.verifyOpts); err != nil {
		return fmt.Errorf("verifying report: %w", err)
	}
	v.logger.Info("Successfully verified report signature")

	// Report content verification.

	v.validateOpts.ReportData = reportData
	if err := validate.SnpAttestation(attestationData, v.validateOpts); err != nil {
		return fmt.Errorf("validating report claims: %w", err)
	}

	//
	// Additional checks.
	//

	// Check for allowed ChipIDs.
	if len(v.allowedChipIDs) != 0 {
		if !slices.ContainsFunc(v.allowedChipIDs, func(id []byte) bool {
			return bytes.Equal(id, attestationData.Report.GetChipId())
		}) {
			return fmt.Errorf("chip ID %x not in allowed chip IDs", attestationData.Report.GetChipId())
		}
	}

	// Report fully validated from here on.

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

// IterativeValidator validates SNP attestation by trying vCPU counts 1–220 until
// one matches the report's measurement. It requires RequireIDBlock to be true in
// the base ValidateOpts and computes the corresponding IDKey hash per iteration.
type IterativeValidator struct {
	seed           [snpmeasure.LaunchDigestSize]byte
	apEIP          uint32
	vcpuSig        uint32
	verifyOpts     *verify.Options
	validateOpts   *validate.Options
	allowedChipIDs [][]byte
	reportSetter   attestation.ReportSetter
	logger         *slog.Logger
	name           string
}

// NewIterativeValidator returns a new IterativeValidator.
// seed is the 1-vCPU launch measurement; apEIP is the AP reset EIP from the OVMF footer.
// vcpuSig is the CPUID signature for the CPU type (e.g. EPYC-Milan, EPYC-Genoa).
// validateOpts must not have Measurement or TrustedIDKeyHashes set — the validator fills them per iteration.
func NewIterativeValidator(
	verifyOpts *verify.Options, validateOpts *validate.Options,
	seed [snpmeasure.LaunchDigestSize]byte, apEIP uint32, vcpuSig uint32,
	allowedChipIDs [][]byte, log *slog.Logger, name string,
) *IterativeValidator {
	return &IterativeValidator{
		seed:           seed,
		apEIP:          apEIP,
		vcpuSig:        vcpuSig,
		verifyOpts:     verifyOpts,
		validateOpts:   validateOpts,
		allowedChipIDs: allowedChipIDs,
		logger:         log,
		name:           name,
	}
}

// NewIterativeValidatorWithReportSetter returns a new IterativeValidator with a report setter.
func NewIterativeValidatorWithReportSetter(
	verifyOpts *verify.Options, validateOpts *validate.Options,
	seed [snpmeasure.LaunchDigestSize]byte, apEIP uint32, vcpuSig uint32,
	allowedChipIDs [][]byte, log *slog.Logger, reportSetter attestation.ReportSetter, name string,
) *IterativeValidator {
	v := NewIterativeValidator(verifyOpts, validateOpts, seed, apEIP, vcpuSig, allowedChipIDs, log, name)
	v.reportSetter = reportSetter
	return v
}

// OID returns the OID for the raw SNP report extension.
func (v *IterativeValidator) OID() asn1.ObjectIdentifier {
	return oid.RawSNPReport
}

// Validate tries vCPU counts 1–220, verifying the attestation once and
// then validating claims against the first matching per-vCPU measurement.
func (v *IterativeValidator) Validate(ctx context.Context, attDocRaw []byte, reportData []byte) (err error) {
	v.logger.Info("Validate called", "name", v.name, "report-data", hex.EncodeToString(reportData))
	defer func() {
		if err != nil {
			v.logger.Debug("Validate failed", "name", v.name, "report-data", hex.EncodeToString(reportData), "error", err)
		} else {
			v.logger.Info("Validate succeeded", "name", v.name, "report-data", hex.EncodeToString(reportData))
		}
	}()

	attestationData := &sevsnp.Attestation{}
	if err := proto.Unmarshal(attDocRaw, attestationData); err != nil {
		return fmt.Errorf("unmarshaling attestation: %w", err)
	}
	if attestationData.Report == nil {
		return fmt.Errorf("attestation missing report")
	}
	v.logger.Debug("Report decoded", "report", protojson.MarshalOptions{Multiline: false}.Format(attestationData.Report))

	if err := addCRLtoVerifyOptions(attestationData, v.verifyOpts); err != nil {
		v.logger.Info("could not use cached CRL from Coordinator aTLS handshake", slog.String("error", err.Error()))
	}

	if err := verify.SnpAttestationContext(ctx, attestationData, v.verifyOpts); err != nil {
		return fmt.Errorf("verifying report: %w", err)
	}
	v.logger.Info("Successfully verified report signature")

	actualMeasurement := attestationData.Report.GetMeasurement()

	for vcpus := 1; vcpus <= 220; vcpus++ {
		expected, err := snpmeasure.ExtendSNPLaunchDigest(v.seed, vcpus, v.apEIP, v.vcpuSig)
		if err != nil {
			return fmt.Errorf("extending launch digest to %d vCPUs: %w", vcpus, err)
		}
		if !bytes.Equal(actualMeasurement, expected[:]) {
			continue
		}

		_, authBlk, err := idblock.IDBlocksFromLaunchDigest(expected, v.validateOpts.GuestPolicy)
		if err != nil {
			return fmt.Errorf("generating ID blocks for %d vCPUs: %w", vcpus, err)
		}
		idKeyBytes, err := authBlk.IDKey.MarshalBinary()
		if err != nil {
			return fmt.Errorf("marshaling IDKey for %d vCPUs: %w", vcpus, err)
		}
		idKeyHash := sha512.Sum384(idKeyBytes)

		opts := *v.validateOpts
		opts.Measurement = expected[:]
		opts.TrustedIDKeyHashes = [][]byte{idKeyHash[:]}
		opts.ReportData = reportData

		if err := validate.SnpAttestation(attestationData, &opts); err != nil {
			return fmt.Errorf("validating report claims: %w", err)
		}

		if len(v.allowedChipIDs) != 0 {
			if !slices.ContainsFunc(v.allowedChipIDs, func(id []byte) bool {
				return bytes.Equal(id, attestationData.Report.GetChipId())
			}) {
				return fmt.Errorf("chip ID %x not in allowed chip IDs", attestationData.Report.GetChipId())
			}
		}

		if v.reportSetter != nil {
			v.reportSetter.SetReport(snpReport{report: attestationData.Report})
		}
		return nil
	}

	return fmt.Errorf("measurement does not match any vCPU count from 1 to 220")
}

// String returns the name as identifier of the validator.
func (v *IterativeValidator) String() string {
	return v.name
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
		return fmt.Errorf("could not parse CRL from attestation data: %w", err)
	}

	// Attestation with v3 report: FMS is set in the report; Product is nil
	// Attestation with v2 report: FMS is 0; Product is set
	var productLine string
	if fms := attestationData.GetReport().GetCpuid1EaxFms(); fms != 0 {
		productLine = kds.ProductLineFromFms(fms)
	} else {
		productLine = kds.ProductLine(attestationData.GetProduct())
	}

	for _, tr := range verifyOpts.TrustedRoots[productLine] {
		tr.CRL = crl
	}

	return nil
}
