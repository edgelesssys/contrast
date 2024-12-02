// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build contrast_unstable_api

package sdk

import (
	"fmt"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/manifest"
)

// ValidatorsFromManifest returns a list of validators corresponding to the reference values in the given manifest.
// Originally an unexported function in the contrast CLI.
// Can be made unexported again, if we decide to move all userapi calls from the CLI to the SDK.
func ValidatorsFromManifest(kdsDir string, m *manifest.Manifest, log *slog.Logger, coordinatorPolicyChecksum []byte) ([]atls.Validator, error) {
	kdsCache := fsstore.New(kdsDir, log.WithGroup("kds-cache"))
	kdsGetter := certcache.NewCachedHTTPSGetter(kdsCache, certcache.NeverGCTicker, log.WithGroup("kds-getter"))

	var validators []atls.Validator

	opts, err := m.SNPValidateOpts(kdsGetter)
	if err != nil {
		return nil, fmt.Errorf("getting SNP validate options: %w", err)
	}
	for _, opt := range opts {
		opt.ValidateOpts.HostData = coordinatorPolicyChecksum
		validators = append(validators, snp.NewValidator(opt.VerifyOpts, opt.ValidateOpts,
			logger.NewWithAttrs(logger.NewNamed(log, "validator"), map[string]string{"tee-type": "snp"}),
		))
	}

	tdxOpts, err := m.TDXValidateOpts()
	if err != nil {
		return nil, fmt.Errorf("generating TDX validation options: %w", err)
	}
	var mrConfigID [48]byte
	copy(mrConfigID[:], coordinatorPolicyChecksum)
	for _, opt := range tdxOpts {
		opt.TdQuoteBodyOptions.MrConfigID = mrConfigID[:]
		validators = append(validators, tdx.NewValidator(&tdx.StaticValidateOptsGenerator{Opts: opt}, logger.NewWithAttrs(logger.NewNamed(log, "validator"), map[string]string{"tee-type": "tdx"})))
	}

	return validators, nil
}
