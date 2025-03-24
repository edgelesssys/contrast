// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/edgelesssys/contrast/internal/idblock"
	"github.com/edgelesssys/contrast/tools/igvm"
	"github.com/google/go-sev-guest/abi"
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
		Use:   "snp-id-block-generator",
		Short: "Generate ID block and ID auth block from a launch digest and guest policy",
		Long:  `Generate ID block and ID auth block from a launch digest and guest policy.`,
		RunE:  runGenerate,
	}

	cmd.Flags().String("launch-digest", "", "Path to launch digest")
	cmd.Flags().String("guest-policy", "", "Path to SNP Guest Policy. Must be JSON file")
	cmd.Flags().String("id-block-out", "", "Output file for ID block")
	cmd.Flags().String("id-auth-out", "", "Output file for ID auth block")
	cmd.Flags().String("id-block-igvm-out", "", "Output file for ID auth block for IGVM")

	must(cmd.MarkFlagRequired("launch-digest"))
	must(cmd.MarkFlagFilename("launch-digest"))

	must(cmd.MarkFlagRequired("guest-policy"))
	must(cmd.MarkFlagFilename("guest-policy"))

	must(cmd.MarkFlagRequired("id-block-out"))
	must(cmd.MarkFlagFilename("id-block-out"))

	must(cmd.MarkFlagRequired("id-auth-out"))
	must(cmd.MarkFlagFilename("id-auth-out"))

	must(cmd.MarkFlagRequired("id-block-igvm-out"))
	must(cmd.MarkFlagFilename("id-block-igvm-out"))

	return cmd
}

func runGenerate(cmd *cobra.Command, _ []string) error {
	// Parse flags
	launchDigestPath, err := cmd.Flags().GetString("launch-digest")
	if err != nil {
		return fmt.Errorf("failed to get launch digest path: %w", err)
	}

	guestPolicyPath, err := cmd.Flags().GetString("guest-policy")
	if err != nil {
		return fmt.Errorf("failed to get guest policy path: %w", err)
	}

	idBlockOutPath, err := cmd.Flags().GetString("id-block-out")
	if err != nil {
		return fmt.Errorf("failed to get id block out path: %w", err)
	}

	idAuthOutPath, err := cmd.Flags().GetString("id-auth-out")
	if err != nil {
		return fmt.Errorf("failed to get id auth out path: %w", err)
	}

	idBlockIGVMOutPath, err := cmd.Flags().GetString("id-block-igvm-out")
	if err != nil {
		return fmt.Errorf("failed to get id block igvm out path: %w", err)
	}

	return generate(afero.NewOsFs(), launchDigestPath, guestPolicyPath, idBlockOutPath, idAuthOutPath, idBlockIGVMOutPath)
}

func generate(fs afero.Fs, launchDigestPath, guestPolicyPath, idBlockOutPath, idAuthOutPath, idBlockIGVMOutPath string) error {
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

	guestPolicyJSONBytes, err := afero.ReadFile(fs, guestPolicyPath)
	if err != nil {
		return fmt.Errorf("failed to read launch digest: %w", err)
	}

	var guestPolicy abi.SnpPolicy
	if err := json.Unmarshal(guestPolicyJSONBytes, &guestPolicy); err != nil {
		return fmt.Errorf("failed to unmarshal guest policy: %w", err)
	}

	idBlk, authBlock, err := idblock.IDBlocksFromLaunchDigest([48]byte(launchDigestBytes), guestPolicy)
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

	igvmID := makeIGVMIDBlock(idBlk, authBlock)
	igvmIDJsonBytes, err := json.Marshal(igvmID)
	if err != nil {
		return fmt.Errorf("failed to marshal igvm id block: %w", err)
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

	igvmIDFile, err := fs.Create(idBlockIGVMOutPath)
	if err != nil {
		return fmt.Errorf("failed to create igvmID file: %w", err)
	}
	defer igvmIDFile.Close()
	if _, err := igvmIDFile.Write(igvmIDJsonBytes); err != nil {
		return fmt.Errorf("failed to write igvmID to file: %w", err)
	}

	return nil
}

func makeIGVMIDBlock(idBlk *idblock.IDBlock, authBlock *idblock.IDAuthentication) *igvm.VhsSnpIDBlock {
	return &igvm.VhsSnpIDBlock{
		CompatibilityMask:  0x1,
		AuthorKeyEnabled:   0x0,
		LD:                 idBlk.LD,
		FamilyID:           idBlk.FamilyID,
		ImageID:            idBlk.ImageID,
		Version:            idBlk.Version,
		GuestSVN:           idBlk.GuestSVN,
		IDKeyAlgorithm:     authBlock.IDKeyAlgo,
		AuthorKeyAlgorithm: authBlock.AuthKeyAlgo,
		IDKeySignature: igvm.VhsSnpIDBlockSignature{
			RComp: authBlock.IDBlockSig.R,
			SComp: authBlock.IDBlockSig.S,
		},
		IDPublicKey: igvm.VhsSnpIDBlockPublicKey{
			Curve: authBlock.IDKey.CurveID,
			QX:    authBlock.IDKey.Qx,
			QY:    authBlock.IDKey.Qy,
		},
		AuthorKeySignature: igvm.VhsSnpIDBlockSignature{
			RComp: authBlock.IDKeySig.R,
			SComp: authBlock.IDKeySig.S,
		},
		AuthorPublicKey: igvm.VhsSnpIDBlockPublicKey{
			Curve: authBlock.AuthKey.CurveID,
			QX:    authBlock.AuthKey.Qx,
			QY:    authBlock.AuthKey.Qy,
		},
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
