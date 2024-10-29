// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/edgelesssys/contrast/cli/telemetry"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	atlsinsecure "github.com/edgelesssys/contrast/internal/attestation/insecure"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/spf13/cobra"
)

const (
	coordHashFilename    = "coordinator-policy.sha256"
	coordRootPEMFilename = "coordinator-root-ca.pem"
	meshCAPEMFilename    = "mesh-ca.pem"
	workloadOwnerPEM     = "workload-owner.pem"
	seedshareOwnerPEM    = "seedshare-owner.pem"
	manifestFilename     = "manifest.json"
	settingsFilename     = "settings.json"
	seedSharesFilename   = "seed-shares.json"
	rulesFilename        = "rules.rego"
	layersCacheFilename  = "layers-cache.json"
	verifyDir            = "verify"
	cacheDirEnv          = "CONTRAST_CACHE_DIR"
)

var (
	// ReleaseImageReplacements contains the image replacements used by contrast.
	//go:embed assets/image-replacements.txt
	ReleaseImageReplacements []byte
	// DefaultCoordinatorPolicyHash is derived from the coordinator release candidate and injected at release build time.
	//
	// It is intentionally left empty for dev builds.
	DefaultCoordinatorPolicyHash = ""
)

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

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func withTelemetry(runFunc func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cmdErr := runFunc(cmd, args)

		if os.Getenv("DO_NOT_TRACK") != "1" {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			cl := telemetry.NewClient()
			_ = cl.SendTelemetry(ctx, cmd, cmdErr)
		}

		return cmdErr
	}
}

// validatorsFromManifest returns a list of validators corresponding to the reference values in the given manifest.
func validatorsFromManifest(m *manifest.Manifest, log *slog.Logger, hostData []byte) ([]atls.Validator, error) {
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

	// TODO(@3u13r): Don't add the insecure validator for all manifests.
	validators = append(validators, atlsinsecure.NewValidator(log))

	return validators, nil
}
