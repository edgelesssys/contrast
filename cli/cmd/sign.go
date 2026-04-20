// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/spf13/cobra"
)

// NewSignCmd creates the contrast sign subcommand.
func NewSignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign [flags]",
		Short: "Sign the given manifest and transition hash",
		Long: `Sign the given manifest and transition hash.

This will read the manifest and previous transition hash and sign them using
the provided workload owner key. The resulting signature can be used for a
signed manifest update at the Coordinator without providing the workload owner
key to the CLI setting the manifest.

Using the prepare flag, the CLI will compute the next transition hash and
output it to a file so it can be signed using an external tool like an HSM.`,
		RunE: withTelemetry(runSign),
	}
	cmd.SetOut(commandOut())

	cmd.Flags().StringP("manifest", "m", manifestFilename, "path to manifest (.json) file")
	cmd.Flags().String("workload-owner-key", workloadOwnerPEM, "path to workload owner key (.pem) file")
	cmd.Flags().String("latest-transition", "", "latest transition hash set at the coordinator (hex string)")
	cmd.Flags().Bool("prepare", false, "prepare the next transition hash for signing without signing it")
	cmd.Flags().String("out", "", "output file for the signature (or next transition hash when using --prepare)")
	must(cmd.MarkFlagRequired("out"))
	must(cmd.MarkFlagFilename("manifest", "json"))

	return cmd
}

func runSign(cmd *cobra.Command, _ []string) error {
	flags, err := parseSignFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	log, err := newCLILogger(cmd)
	if err != nil {
		return err
	}

	manifestBytes, err := os.ReadFile(flags.manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}
	var m manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return fmt.Errorf("validating manifest: %w", err)
	}

	if flags.latestTransition == "" {
		data, err := os.ReadFile(filepath.Join(flags.workspaceDir, verifyDir, latestTransitionHashFilename))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("reading previous transition hash: %w", err)
		} else if errors.Is(err, os.ErrNotExist) {
			data = []byte(strings.Repeat("00", history.HashSize)) // Assume initial set manifest
		}
		flags.latestTransition = string(data)
	}
	previousTransitionHash, err := hex.DecodeString(flags.latestTransition)
	if err != nil {
		return fmt.Errorf("decoding latest transition hash: %w", err)
	}
	if len(previousTransitionHash) != history.HashSize {
		return fmt.Errorf("invalid latest transition hash byte length: got %d, want %d", len(previousTransitionHash), history.HashSize)
	}

	tr := &history.Transition{
		ManifestHash:           history.Digest(manifestBytes),
		PreviousTransitionHash: [history.HashSize]byte(previousTransitionHash),
	}
	transitionHash := tr.Digest()
	transitionHashHex := hex.AppendEncode(nil, transitionHash[:])

	if flags.prepare {
		if err := os.WriteFile(flags.out, transitionHashHex, 0o644); err != nil {
			return fmt.Errorf("writing next transition hash to file: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Next transition hash written to %s.\n", flags.out)
		return nil
	}

	workloadOwnerKey, err := loadWorkloadOwnerKey(flags.workloadOwnerKeyPath, &m, log)
	if err != nil {
		return fmt.Errorf("loading workload owner key: %w", err)
	}

	signingHash := sha256.Sum256(transitionHashHex)
	sig, err := ecdsa.SignASN1(rand.Reader, workloadOwnerKey, signingHash[:])
	if err != nil {
		return fmt.Errorf("signing transition hash: %w", err)
	}

	if err := os.WriteFile(flags.out, sig, 0o644); err != nil {
		return fmt.Errorf("writing signature to file: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Transition hash signed and signature written to %s.\n", flags.out)

	return nil
}

type signFlags struct {
	manifestPath         string
	workloadOwnerKeyPath string
	latestTransition     string
	prepare              bool
	out                  string
	workspaceDir         string
}

func parseSignFlags(cmd *cobra.Command) (*signFlags, error) {
	flags := &signFlags{}
	var err error

	flags.manifestPath, err = cmd.Flags().GetString("manifest")
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest flag: %w", err)
	}
	flags.workloadOwnerKeyPath, err = cmd.Flags().GetString("workload-owner-key")
	if err != nil {
		return nil, fmt.Errorf("getting workload-owner-key flag: %w", err)
	}
	flags.latestTransition, err = cmd.Flags().GetString("latest-transition")
	if err != nil {
		return nil, fmt.Errorf("getting latest-transition flag: %w", err)
	}
	flags.prepare, err = cmd.Flags().GetBool("prepare")
	if err != nil {
		return nil, fmt.Errorf("getting prepare flag: %w", err)
	}
	flags.out, err = cmd.Flags().GetString("out")
	if err != nil {
		return nil, fmt.Errorf("getting dry-run flag: %w", err)
	}
	flags.workspaceDir, err = cmd.Flags().GetString("workspace-dir")
	if err != nil {
		return nil, fmt.Errorf("getting workspace-dir flag: %w", err)
	}

	if flags.workspaceDir != "" {
		// Prepend default paths with workspaceDir
		if !cmd.Flags().Changed("manifest") {
			flags.manifestPath = path.Join(flags.workspaceDir, flags.manifestPath)
		}
		if !cmd.Flags().Changed("workload-owner-key") {
			flags.workloadOwnerKeyPath = path.Join(flags.workspaceDir, flags.workloadOwnerKeyPath)
		}
	}

	return flags, nil
}
