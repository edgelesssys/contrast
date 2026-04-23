// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"strings"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/manifest"
	snpmeasure "github.com/edgelesssys/contrast/internal/snp"
)

// ValidatorsFromManifest returns a list of validators corresponding to the reference values in the given manifest.
// Originally an unexported function in the contrast CLI.
// Can be made unexported again, if we decide to move all userapi calls from the CLI to the SDK.
// Validators MUST NOT be used concurrently.
func ValidatorsFromManifest(kdsGetter *certcache.CachedHTTPSGetter, m *manifest.Manifest, log *slog.Logger) ([]atls.Validator, error) {
	var validators []atls.Validator

	coordPolicyHash, err := m.CoordinatorPolicyHash()
	if err != nil {
		return nil, fmt.Errorf("getting coordinator policy hash: %w", err)
	}
	coordPolicyHashBytes, err := coordPolicyHash.Bytes()
	if err != nil {
		return nil, fmt.Errorf("converting coordinator policy hash to bytes: %w", err)
	}
	opts, err := m.SNPValidateOpts(kdsGetter)
	if err != nil {
		return nil, fmt.Errorf("getting SNP validate options: %w", err)
	}
	for i, opt := range opts {
		opt.ValidateOpts.HostData = coordPolicyHashBytes
		name := fmt.Sprintf("snp-%d-%s", i, strings.TrimPrefix(opt.VerifyOpts.Product.Name.String(), "SEV_PRODUCT_"))
		validatorLog := logger.NewWithAttrs(logger.NewNamed(log, "validator"), map[string]string{"reference-values": name})
		var v atls.Validator
		if len(opt.APEIP) == 4 {
			seed := [snpmeasure.LaunchDigestSize]byte(opt.ValidateOpts.Measurement)
			apEIP := binary.BigEndian.Uint32(opt.APEIP)
			v = snp.NewIterativeValidator(opt.VerifyOpts, opt.ValidateOpts, seed, apEIP, opt.VCPUSig, opt.AllowedChipIDs, validatorLog, name)
		} else {
			v = snp.NewValidator(opt.VerifyOpts, opt.ValidateOpts, opt.AllowedChipIDs, validatorLog, name)
		}
		validators = append(validators, v)
	}

	tdxOpts, err := m.TDXValidateOpts(kdsGetter)
	if err != nil {
		return nil, fmt.Errorf("generating TDX validation options: %w", err)
	}
	var mrConfigID [48]byte
	copy(mrConfigID[:], coordPolicyHashBytes)
	for i, opt := range tdxOpts {
		name := fmt.Sprintf("tdx-%d", i)
		opt.ValidateOpts.TdQuoteBodyOptions.MrConfigID = mrConfigID[:]
		validators = append(validators, tdx.NewValidator(opt.VerifyOpts, &tdx.StaticValidateOptsGenerator{Opts: opt.ValidateOpts}, opt.AllowedPIIDs, logger.NewWithAttrs(logger.NewNamed(log, "validator"), map[string]string{"reference-values": name}), name))
	}

	return validators, nil
}
