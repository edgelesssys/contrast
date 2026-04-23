// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/edgelesssys/contrast/cli/telemetry"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	coordRootPEMFilename         = "coordinator-root-ca.pem"
	meshCAPEMFilename            = "mesh-ca.pem"
	workloadOwnerPEM             = "workload-owner.pem"
	seedshareOwnerPEM            = "seedshare-owner.pem"
	manifestFilename             = "manifest.json"
	settingsFilename             = "settings.json"
	seedSharesFilename           = "seed-shares.json"
	rulesFilename                = "rules.rego"
	layersCacheFilename          = "layers-cache.json"
	latestTransitionHashFilename = "latest-transition"
	historyFilename              = "history.yml"
	verifyDir                    = "verify"
)

// ReleaseImageReplacements contains the image replacements used by contrast.
//
//go:embed assets/image-replacements.txt
var ReleaseImageReplacements []byte

// snpAPEIPHex contains the SEV-ES AP reset EIP read from the OVMF firmware at build time,
// encoded as an 8-character lowercase hex string (4 bytes, big-endian).
//
//go:embed assets/ap-eip.hex
var snpAPEIPHex []byte

// parsedAPEIP parses the AP reset EIP embedded at build time and returns it as a uint32.
func parsedAPEIP() (uint32, error) {
	b, err := hex.DecodeString(strings.TrimSpace(string(snpAPEIPHex)))
	if err != nil {
		return 0, fmt.Errorf("invalid ap-eip.hex: %w", err)
	}
	if len(b) != 4 {
		return 0, fmt.Errorf("invalid ap-eip.hex: expected 4 bytes, got %d", len(b))
	}
	return binary.BigEndian.Uint32(b), nil
}

func commandOut() io.Writer {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return nil // use out writer of parent
	}
	return io.Discard
}

func cachedir(subdir string) (string, error) {
	dir := os.Getenv(constants.CacheDirEnvVar)
	if dir == "" {
		cachedir, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(cachedir, "contrast")
	}
	return filepath.Join(dir, subdir), nil
}

func cachedHTTPSGetter(log *slog.Logger) (*certcache.CachedHTTPSGetter, error) {
	kdsDir, err := cachedir("kds")
	if err != nil {
		return nil, fmt.Errorf("getting cache dir: %w", err)
	}
	log.Debug("Using KDS cache dir", "dir", kdsDir)

	kdsCache := fsstore.New(afero.NewBasePathFs(afero.NewOsFs(), kdsDir), log.WithGroup("kds-cache"))
	return certcache.NewCachedHTTPSGetter(kdsCache, certcache.NeverGCTicker, log.WithGroup("kds-getter")), nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// golangci-lint complains complains that ctx is not passed through the NewVerifyCmd to the withTelemetry function.
// This is a false positive, since withTelemetry only returns a function. The function is passed a cobra.Command
// and the cmd.Context from that is used when the actual function executes.
// Moreover, contextcheck only throws an error if the it checks the module with the e2e build tag, therefore
// we need to disable the nolintlint linter also.
//
//nolint:contextcheck // similar to https://github.com/kkHAIKE/contextcheck/issues/24
//nolint:nolintlint
func withTelemetry(runFunc func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cmdErr := runFunc(cmd, args)

		if os.Getenv("DO_NOT_TRACK") != "1" {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Second*5)
			defer cancel()

			cl := telemetry.NewClient()
			_ = cl.SendTelemetry(ctx, cmd, cmdErr)
		}

		return cmdErr
	}
}
