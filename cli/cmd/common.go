// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/edgelesssys/contrast/cli/telemetry"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
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
	//go:embed assets/genpolicy
	genpolicyBin []byte
	//go:embed assets/genpolicy-settings.json
	defaultGenpolicySettings []byte
	//go:embed assets/genpolicy-rules.rego
	defaultRules []byte
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

// dialCoordinatorWithKey establishes an attested gRPC connection to the coordinator at the given endpoint,
// verifying the attestation report against the given private key.
//
// The resulting connection must be closed by the caller.
func dialCoordinatorWithKey(ctx context.Context, log *slog.Logger, m manifest.Manifest, hostData []byte,
	coordinatorEndpoint, workloadOwnerKeyPath string,
) (*grpc.ClientConn, error) {
	workloadOwnerKey, err := loadWorkloadOwnerKey(workloadOwnerKeyPath, nil, log)
	if err != nil {
		return nil, fmt.Errorf("loading workload owner key: %w", err)
	}

	validators, err := localValidators(log, m, hostData)
	if err != nil {
		return nil, fmt.Errorf("getting local validators: %w", err)
	}

	dialer := dialer.NewWithKey(atls.NoIssuer, validators, &net.Dialer{}, workloadOwnerKey)

	log.Debug("Dialing coordinator", "endpoint", coordinatorEndpoint)
	conn, err := dialer.Dial(ctx, coordinatorEndpoint)
	if err != nil {
		return nil, fmt.Errorf("dialing coordinator: %w", err)
	}

	return conn, nil
}

// localValidators returns a list of validators according to the given manifest and the local certificate
// cache directory.
func localValidators(log *slog.Logger, m manifest.Manifest, hostData []byte) ([]atls.Validator, error) {
	certCacheDir, err := cachedir("certs")
	if err != nil {
		return nil, fmt.Errorf("getting cache dir: %w", err)
	}
	log.Debug("Using certificate cache dir", "dir", certCacheDir)

	validateOptsGen, err := newCoordinatorValidateOptsGen(m, hostData)
	if err != nil {
		return nil, fmt.Errorf("generating validate opts: %w", err)
	}
	kdsCache := fsstore.New(certCacheDir, log.WithGroup("kds-cache"))
	kdsGetter := snp.NewCachedHTTPSGetter(kdsCache, snp.NeverGCTicker, log.WithGroup("kds-getter"))
	validator := snp.NewValidator(validateOptsGen, kdsGetter,
		logger.NewWithAttrs(logger.NewNamed(log, "validator"), map[string]string{"tee-type": "snp"}),
	)

	return []atls.Validator{validator}, nil
}
