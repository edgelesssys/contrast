// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/edgelesssys/contrast/cli/telemetry"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
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
	// CoordinatorPolicyHashesFallback are derived from the coordinator release candidate and injected at release build time.
	//
	// It is intentionally left empty for dev builds.
	//go:embed assets/coordinator-policy-hashes-fallback.json
	CoordinatorPolicyHashesFallback []byte
)

type coordinatorPolicyHashes map[platforms.Platform]manifest.HexString

// defaultCoordinatorPolicyHash returns the default coordinator policy hash for the given platform.
func defaultCoordinatorPolicyHash(p platforms.Platform) (manifest.HexString, error) {
	defaults := make(coordinatorPolicyHashes)
	if err := json.Unmarshal(CoordinatorPolicyHashesFallback, &defaults); err != nil {
		return "", fmt.Errorf("unmarshaling coordinator policy hashes fallback: %w", err)
	}
	defaultHash, ok := defaults[p]
	if !ok {
		return "", fmt.Errorf("no default coordinator policy hash for %s", p)
	}
	return defaultHash, nil
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
