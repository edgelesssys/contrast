// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"time"

	"github.com/edgelesssys/contrast/cli/telemetry"
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
