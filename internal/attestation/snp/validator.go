// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package snp

import (
	"context"
	_ "embed"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/attestation/reportdata"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
	"github.com/google/go-sev-guest/verify"
	"github.com/google/go-sev-guest/verify/trust"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/proto"
)

// Validator validates attestation statements.
type Validator struct {
	validateOptsGen validateOptsGenerator
	callbackers     []validateCallbacker
	kdsGetter       trust.HTTPSGetter
	logger          *slog.Logger
	metrics         metrics
}

type metrics struct {
	attestationFailures prometheus.Counter
}

type validateCallbacker interface {
	ValidateCallback(ctx context.Context, report *sevsnp.Report, validatorOID asn1.ObjectIdentifier,
		reportRaw, nonce, peerPublicKey []byte) error
}

type validateOptsGenerator interface {
	SNPValidateOpts(report *sevsnp.Report) (*validate.Options, error)
}

// StaticValidateOptsGenerator returns validate.Options generator that returns
// static validation options.
type StaticValidateOptsGenerator struct {
	Opts *validate.Options
}

// SNPValidateOpts return the SNP validation options.
func (v *StaticValidateOptsGenerator) SNPValidateOpts(_ *sevsnp.Report) (*validate.Options, error) {
	return v.Opts, nil
}

// NewValidator returns a new Validator.
func NewValidator(optsGen validateOptsGenerator, kdsGetter trust.HTTPSGetter, log *slog.Logger) *Validator {
	return &Validator{
		validateOptsGen: optsGen,
		kdsGetter:       kdsGetter,
		logger:          log,
	}
}

// NewValidatorWithCallbacks returns a new Validator with callbacks.
func NewValidatorWithCallbacks(optsGen validateOptsGenerator, kdsGetter trust.HTTPSGetter, log *slog.Logger, attestataionFailures prometheus.Counter, callbacks ...validateCallbacker) *Validator {
	return &Validator{
		validateOptsGen: optsGen,
		callbackers:     callbacks,
		kdsGetter:       kdsGetter,
		logger:          log,
		metrics:         metrics{attestationFailures: attestataionFailures},
	}
}

// OID returns the OID of the validator.
func (v *Validator) OID() asn1.ObjectIdentifier {
	return oid.RawSNPReport
}

// Validate a TPM based attestation.
func (v *Validator) Validate(ctx context.Context, attDocRaw []byte, nonce []byte, peerPublicKey []byte) (err error) {
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

	attestation := &sevsnp.Attestation{}
	if err := proto.Unmarshal(attDocRaw, attestation); err != nil {
		return fmt.Errorf("unmarshalling attestation: %w", err)
	}

	if attestation.Report == nil {
		return fmt.Errorf("attestation missing report")
	}
	reportRaw, err := abi.ReportToAbiBytes(attestation.Report)
	if err != nil {
		return fmt.Errorf("converting report to abi: %w", err)
	}
	v.logger.Info("Report decoded", "reportRaw", hex.EncodeToString(reportRaw))

	verifyOpts := verify.DefaultOptions()
	// TODO(Freax13): We won't need this once https://github.com/google/go-sev-guest/pull/127 is merged.
	verifyOpts.TrustedRoots = trustedRoots()
	verifyOpts.Product = attestation.Product
	verifyOpts.CheckRevocations = true
	verifyOpts.Getter = v.kdsGetter

	// Report signature verification.

	if err := verify.SnpAttestation(attestation, verifyOpts); err != nil {
		return fmt.Errorf("verifying report: %w", err)
	}
	v.logger.Info("Successfully verified report signature")

	// Validate the report data.

	reportDataExpected := reportdata.Construct(peerPublicKey, nonce)
	validateOpts, err := v.validateOptsGen.SNPValidateOpts(attestation.Report)
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
		if err := callbacker.ValidateCallback(
			ctx, attestation.Report, v.OID(), reportRaw, nonce, peerPublicKey,
		); err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	v.logger.Info("Validate finished successfully")
	return nil
}

var (
	// source: https://kdsintf.amd.com/vcek/v1/Milan/cert_chain
	//go:embed Milan.pem
	askArkMilanVcekBytes []byte
	// source: https://kdsintf.amd.com/vcek/v1/Genoa/cert_chain
	//go:embed Genoa.pem
	askArkGenoaVcekBytes []byte
)

func trustedRoots() map[string][]*trust.AMDRootCerts {
	trustedRoots := make(map[string][]*trust.AMDRootCerts)

	milanCerts := trust.AMDRootCertsProduct("Milan")
	if err := milanCerts.FromKDSCertBytes(askArkMilanVcekBytes); err != nil {
		panic(fmt.Errorf("failed to parse cert: %w", err))
	}
	trustedRoots["Milan"] = []*trust.AMDRootCerts{milanCerts}

	genoaCerts := trust.AMDRootCertsProduct("Genoa")
	if err := genoaCerts.FromKDSCertBytes(askArkGenoaVcekBytes); err != nil {
		panic(fmt.Errorf("failed to parse cert: %w", err))
	}
	trustedRoots["Genoa"] = []*trust.AMDRootCerts{genoaCerts}

	return trustedRoots
}
