// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/spf13/cobra"
)

// NewRecoverCmd creates the contrast recover subcommand.
func NewRecoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recover [flags]",
		Short: "recover a contrast deployment after restart",
		Long: `Recover a contrast deployment after restart.

The state of the Coordinator is stored protected on a persistent volume.
After a restart, the Coordinator requires the seed to derive the signing
key and verify the state integrity.

The recover command is used to provide the seed to the Coordinator.`,
		RunE: withTelemetry(runRecover),
	}

	cmd.Flags().StringP("manifest", "m", manifestFilename, "path to manifest (.json) file")
	cmd.Flags().StringP("coordinator", "c", "", "endpoint the coordinator can be reached at")
	must(cobra.MarkFlagRequired(cmd.Flags(), "coordinator"))
	cmd.Flags().String("coordinator-policy-hash", DefaultCoordinatorPolicyHash, "override the expected policy hash of the coordinator")
	cmd.Flags().String("workload-owner-key", workloadOwnerPEM,
		"path to workload owner key (.pem) file (can be passed more than once)")
	cmd.Flags().String("seedshare-owner-key", seedshareOwnerPEM, "private key file to decrypt the seed share")
	cmd.Flags().String("seed", seedSharesFilename, "file with the encrypted seed shares")

	return cmd
}

func runRecover(cmd *cobra.Command, _ []string) error {
	flags, err := parseRecoverFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	log, err := newCLILogger(cmd)
	if err != nil {
		return err
	}
	log.Debug("Starting recovery")

	manifestBytes, err := os.ReadFile(flags.manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}
	var m manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}
	workloadOwnerKey, err := loadWorkloadOwnerKey(flags.workloadOwnerKeyPath, nil, log)
	if err != nil {
		return fmt.Errorf("loading workload owner key: %w", err)
	}
	seed, salt, err := decryptedSeedFromShares(flags.seedSharesFilename, flags.seedShareOwnerKeyPath)
	if err != nil {
		return fmt.Errorf("decrypting seed: %w", err)
	}

	kdsDir, err := cachedir("kds")
	if err != nil {
		return fmt.Errorf("getting cache dir: %w", err)
	}
	log.Debug("Using KDS cache dir", "dir", kdsDir)

	validateOptsGen, err := newCoordinatorValidateOptsGen(m, flags.policy)
	if err != nil {
		return fmt.Errorf("generating validate opts: %w", err)
	}
	kdsCache := fsstore.New(kdsDir, log.WithGroup("kds-cache"))
	kdsGetter := snp.NewCachedHTTPSGetter(kdsCache, snp.NeverGCTicker, log.WithGroup("kds-getter"))
	validator := snp.NewValidator(validateOptsGen, kdsGetter,
		logger.NewWithAttrs(logger.NewNamed(log, "validator"), map[string]string{"tee-type": "snp"}),
	)
	dialer := dialer.NewWithKey(atls.NoIssuer, validator, &net.Dialer{}, workloadOwnerKey)

	log.Debug("Dialing coordinator", "endpoint", flags.coordinator)
	conn, err := dialer.Dial(cmd.Context(), flags.coordinator)
	if err != nil {
		return fmt.Errorf("dialing coordinator: %w", err)
	}
	defer conn.Close()

	client := userapi.NewUserAPIClient(conn)
	req := &userapi.RecoverRequest{
		Seed: seed,
		Salt: salt,
	}
	if _, err := client.Recover(cmd.Context(), req); err != nil {
		return fmt.Errorf("recovering: %w", err)
	}
	log.Debug("Got response")

	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Successfully recovered the Coordinator")
	return nil
}

type recoverFlags struct {
	coordinator           string
	policy                []byte
	seedSharesFilename    string
	seedShareOwnerKeyPath string
	workloadOwnerKeyPath  string
	manifestPath          string
	workspaceDir          string
}

func decryptedSeedFromShares(seedSharesPath, seedShareOwnerKeyPath string) ([]byte, []byte, error) {
	keyBytes, err := os.ReadFile(seedShareOwnerKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading seedshare owner key: %w", err)
	}
	key, err := manifest.ParseSeedshareOwnerPrivateKey(keyBytes)
	if err != nil {
		return nil, nil, err
	}
	pubHexStr := manifest.MarshalSeedShareOwnerKey(&key.PublicKey).String()
	var seedShareDoc userapi.SeedShareDocument
	seedShareBytes, err := os.ReadFile(seedSharesPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading seed shares: %w", err)
	}
	if err := json.Unmarshal(seedShareBytes, &seedShareDoc); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling seed shares: %w", err)
	}
	for _, share := range seedShareDoc.SeedShares {
		if share.PublicKey != pubHexStr {
			continue
		}
		seed, err := manifest.DecryptSeedShare(key, share)
		if err != nil {
			return nil, nil, fmt.Errorf("decrypting seed share: %w", err)
		}
		return seed, seedShareDoc.Salt, nil
	}
	return nil, nil, fmt.Errorf("no matching seed share found")
}

func parseRecoverFlags(cmd *cobra.Command) (*recoverFlags, error) {
	coordinator, err := cmd.Flags().GetString("coordinator")
	if err != nil {
		return nil, err
	}
	policy, err := decodeCoordinatorPolicyHash(cmd.Flags())
	if err != nil {
		return nil, err
	}
	seed, err := cmd.Flags().GetString("seed")
	if err != nil {
		return nil, err
	}
	seedShareOwnerKeyPath, err := cmd.Flags().GetString("seedshare-owner-key")
	if err != nil {
		return nil, err
	}
	workloadOwnerKeyPath, err := cmd.Flags().GetString("workload-owner-key")
	if err != nil {
		return nil, err
	}
	manifestPath, err := cmd.Flags().GetString("manifest")
	if err != nil {
		return nil, err
	}
	workspaceDir, err := cmd.Flags().GetString("workspace-dir")
	if err != nil {
		return nil, err
	}

	if workspaceDir != "" {
		// Prepend default paths with workspaceDir
		if !cmd.Flags().Changed("manifest") {
			manifestPath = filepath.Join(workspaceDir, manifestFilename)
		}
		if !cmd.Flags().Changed("workload-owner-key") {
			workloadOwnerKeyPath = filepath.Join(workspaceDir, workloadOwnerKeyPath)
		}
		if !cmd.Flags().Changed("seedshare-owner-key") {
			seedShareOwnerKeyPath = filepath.Join(workspaceDir, seedShareOwnerKeyPath)
		}
		if !cmd.Flags().Changed("seed") {
			seed = filepath.Join(workspaceDir, seedSharesFilename)
		}
	}

	return &recoverFlags{
		coordinator:           coordinator,
		policy:                policy,
		seedSharesFilename:    seed,
		seedShareOwnerKeyPath: seedShareOwnerKeyPath,
		workloadOwnerKeyPath:  workloadOwnerKeyPath,
		manifestPath:          manifestPath,
		workspaceDir:          workspaceDir,
	}, nil
}
