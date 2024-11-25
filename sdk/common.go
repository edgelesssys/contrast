// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package sdk

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/manifest"
)

const cacheDirEnv = "CONTRAST_CACHE_DIR"

// ValidatorsFromManifest returns a list of validators corresponding to the reference values in the given manifest.
// Originally an unexported function in the contrast CLI.
// Can be made unexported again, if we decide to move all userapi calls from the CLI to the SDK.
func ValidatorsFromManifest(m *manifest.Manifest, log *slog.Logger, hostData []byte) ([]atls.Validator, error) {
	kdsDir, err := cachedir("kds")
	if err != nil {
		return nil, fmt.Errorf("getting cache dir: %w", err)
	}
	log.Debug("Using KDS cache dir", "dir", kdsDir)
	kdsCache := fsstore.New(kdsDir, log.WithGroup("kds-cache"))
	kdsGetter := certcache.NewCachedHTTPSGetter(kdsCache, certcache.NeverGCTicker, log.WithGroup("kds-getter"))

	var validators []atls.Validator

	opts, err := m.SNPValidateOpts(kdsGetter)
	if err != nil {
		return nil, fmt.Errorf("getting SNP validate options: %w", err)
	}
	for _, opt := range opts {
		opt.ValidateOpts.HostData = hostData
		validators = append(validators, snp.NewValidator(opt.VerifyOpts, opt.ValidateOpts,
			logger.NewWithAttrs(logger.NewNamed(log, "validator"), map[string]string{"tee-type": "snp"}),
		))
	}

	tdxOpts, err := m.TDXValidateOpts()
	if err != nil {
		return nil, fmt.Errorf("generating TDX validation options: %w", err)
	}
	var mrConfigID [48]byte
	copy(mrConfigID[:], hostData)
	for _, opt := range tdxOpts {
		opt.TdQuoteBodyOptions.MrConfigID = mrConfigID[:]
		validators = append(validators, tdx.NewValidator(&tdx.StaticValidateOptsGenerator{Opts: opt}, logger.NewWithAttrs(logger.NewNamed(log, "validator"), map[string]string{"tee-type": "tdx"})))
	}

	return validators, nil
}

func cachedir(subdir string) (string, error) {
	dir := os.Getenv(cacheDirEnv)
	if dir == "" {
		cachedir, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(cachedir, "contrast")
	}
	return filepath.Join(dir, subdir), nil
}
