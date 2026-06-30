// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"context"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/internal/atls/validators"
	"github.com/edgelesssys/contrast/internal/attestation"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/oid"
	snpmeasure "github.com/edgelesssys/contrast/internal/snp"
)

// Validator creates a validator that only succeeds for workloads whose policy is in the manifest.
//
// The validator MUST NOT be used concurrently, which is a limitation of the wrapped SNP validator.
func (m *Manifest) Validator(log *slog.Logger, kdsGetter *certcache.CachedHTTPSGetter, reportSetter attestation.ReportSetter) (validators.Validator, error) {
	var allValidators []validators.Validator

	snpOpts, err := m.SNPValidateOpts(kdsGetter)
	if err != nil {
		log.Error("Could not generate SNP validation options", "error", err)
		return nil, fmt.Errorf("generating SNP validation options: %w", err)
	}

	for i, opt := range snpOpts {
		name := fmt.Sprintf("snp-%d-%s", i, strings.TrimPrefix(opt.VerifyOpts.Product.Name.String(), "SEV_PRODUCT_"))
		validatorLog := logger.NewWithAttrs(logger.NewNamed(log, "validator"), map[string]string{"reference-values": name})
		var validator validators.Validator
		if len(opt.APEIP) == 4 {
			seed := [snpmeasure.LaunchDigestSize]byte(opt.ValidateOpts.Measurement)
			apEIP := binary.BigEndian.Uint32(opt.APEIP)
			validator = snp.NewIterativeValidatorWithReportSetter(opt.VerifyOpts, opt.ValidateOpts, seed, apEIP, opt.VCPUSig, opt.AllowedChipIDs, validatorLog, reportSetter, name)
		} else {
			validator = snp.NewValidatorWithReportSetter(opt.VerifyOpts, opt.ValidateOpts, opt.AllowedChipIDs, validatorLog, reportSetter, name)
		}
		allValidators = append(allValidators, validators.WithFixedOID(oid.RawSNPReport, validator))
	}

	tdxOpts, err := m.TDXValidateOpts(kdsGetter)
	if err != nil {
		log.Error("Could not generate TDX validation options", "error", err)
		return nil, fmt.Errorf("generating TDX validation options: %w", err)
	}
	for i, opt := range tdxOpts {
		name := fmt.Sprintf("tdx-%d", i)
		validator := tdx.NewValidatorWithReportSetter(opt.VerifyOpts, &tdx.StaticValidateOptsGenerator{Opts: opt.ValidateOpts}, opt.AllowedPIIDs,
			logger.NewWithAttrs(logger.NewNamed(log, "validator"), map[string]string{"reference-values": name}), reportSetter, name)
		allValidators = append(allValidators, validators.WithFixedOID(oid.RawTDXReport, validator))
	}

	return validators.Any(allValidators...), nil
}

// ErrWrongCoordinatorPolicyHash is returned when the Coordinator policy hash does not match the manifest policy hash.
var ErrWrongCoordinatorPolicyHash = errors.New("wrong policy hash for Coordinator")

// CoordinatorValidator returns a validator that succeeds only for workloads with the Coordinator role.
//
// This is a more restrictive version of Validator, see the warning there.
func (m *Manifest) CoordinatorValidator(log *slog.Logger, kdsGetter *certcache.CachedHTTPSGetter) (validators.Validator, error) {
	coordPolicyHashes, err := m.CoordinatorPolicyHashes()
	if err != nil {
		return nil, fmt.Errorf("getting coordinator policy hash: %w", err)
	}
	return coordinatorValidator(m.Validator, coordPolicyHashes, log, kdsGetter)
}

type validatorFactory func(log *slog.Logger, kdsGetter *certcache.CachedHTTPSGetter, reportSetter attestation.ReportSetter) (validators.Validator, error)

// coordinatorValidator is the implementation of the post-validation step for CoordinatorValidator.
// It can be parametrized for testing.
func coordinatorValidator(validatorFactory validatorFactory, coordPolicyHashes []HexString, log *slog.Logger, kdsGetter *certcache.CachedHTTPSGetter) (validators.Validator, error) {
	var allHashes [][]byte
	for _, hash := range coordPolicyHashes {
		b, err := hash.Bytes()
		if err != nil {
			return nil, fmt.Errorf("converting coordinator policy hash %q to bytes: %w", hash.String(), err)
		}
		allHashes = append(allHashes, b)
	}

	return validators.ValidatorFunc(func(ctx context.Context, oid asn1.ObjectIdentifier, attDoc []byte, reportData []byte) error {
		// We're creating the validator here to make execution reentrant and thread-safe. This way,
		// the validators and the captured report variable are not shared. Constructing the
		// validators is anyway orders of magnitude faster than doing the validation.
		var report attestation.Report
		validator, err := validatorFactory(log, kdsGetter, attestation.ReportSetterFunc(func(r attestation.Report) {
			report = r
		}))
		if err != nil {
			return fmt.Errorf("creating validator from manifest: %w", err)
		}
		if err := validator.Validate(ctx, oid, attDoc, reportData); err != nil {
			return err
		}
		if !slices.ContainsFunc(allHashes, func(b []byte) bool {
			return slices.Equal(report.HostData(), b)
		}) {
			return fmt.Errorf("%w: got %x, want %v", ErrWrongCoordinatorPolicyHash, report.HostData(), coordPolicyHashes)
		}
		return nil
	}), nil
}
