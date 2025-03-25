// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/edgelesssys/contrast/internal/idblock"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func main() {
	if err := execute(); err != nil {
		os.Exit(1)
	}
}

func execute() error {
	cmd := newRootCmd()
	return cmd.ExecuteContext(context.Background())
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snp-qemu-id-block [path to launch digest]",
		Short: "Generate ID block and ID auth block from a launch digest",
		Long:  `Generate ID block and ID auth block from a launch digest.`,
		RunE:  runGenerate,
	}

	cmd.Flags().String("launch-digest", "", "Path to launch digest")
	cmd.Flags().String("id-block-out", "", "Output file for ID block")
	cmd.Flags().String("id-auth-out", "", "Output file for ID auth block")

	must(cmd.MarkFlagRequired("launch-digest"))
	must(cmd.MarkFlagFilename("launch-digest"))

	must(cmd.MarkFlagRequired("id-block-out"))
	must(cmd.MarkFlagFilename("id-block-out"))

	must(cmd.MarkFlagRequired("id-auth-out"))
	must(cmd.MarkFlagFilename("id-auth-out"))

	return cmd
}

func runGenerate(cmd *cobra.Command, _ []string) error {
	// Parse flags
	launchDigestPath, err := cmd.Flags().GetString("launch-digest")
	if err != nil {
		return fmt.Errorf("failed to get launch digest path: %w", err)
	}

	idBlockOutPath, err := cmd.Flags().GetString("id-block-out")
	if err != nil {
		return fmt.Errorf("failed to get id block out path: %w", err)
	}

	idAuthOutPath, err := cmd.Flags().GetString("id-auth-out")
	if err != nil {
		return fmt.Errorf("failed to get id auth out path: %w", err)
	}

	return generate(afero.NewOsFs(), launchDigestPath, idBlockOutPath, idAuthOutPath)
}

func generate(fs afero.Fs, launchDigestPath, idBlockOutPath, idAuthOutPath string) error {
	launchDigest, err := afero.ReadFile(fs, launchDigestPath)
	if err != nil {
		return fmt.Errorf("failed to read launch digest: %w", err)
	}

	launchDigestBytes, err := hex.DecodeString(string(launchDigest))
	if err != nil {
		return fmt.Errorf("failed to decode launch digest: %w", err)
	}
	if len(launchDigestBytes) != 48 {
		return fmt.Errorf("launch digest must be 48 bytes, got %d", len(launchDigestBytes))
	}

	idBlk, authBlock, err := idblock.IDBlocksFromLaunchDigest([48]byte(launchDigestBytes))
	if err != nil {
		return fmt.Errorf("failed to generate ID blocks: %w", err)
	}

	idAuthBytes, err := authBlock.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal idAuth block: %w", err)
	}
	idBlkBytes, err := idBlk.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal id block: %w", err)
	}

	idBlockFile, err := fs.Create(idBlockOutPath)
	if err != nil {
		return fmt.Errorf("failed to create idBlock file: %w", err)
	}
	defer idBlockFile.Close()
	if _, err := idBlockFile.Write([]byte(base64.StdEncoding.EncodeToString(idBlkBytes))); err != nil {
		return fmt.Errorf("failed to write idBlock to file: %w", err)
	}

	authBlockFile, err := fs.Create(idAuthOutPath)
	if err != nil {
		return fmt.Errorf("failed to create authBlock file: %w", err)
	}
	defer authBlockFile.Close()
	if _, err := authBlockFile.Write([]byte(base64.StdEncoding.EncodeToString(idAuthBytes))); err != nil {
		return fmt.Errorf("failed to write authBlock to file: %w", err)
	}

	return nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
