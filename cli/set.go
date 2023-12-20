package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/edgelesssys/nunki/internal/atls"
	"github.com/edgelesssys/nunki/internal/attestation/snp"
	"github.com/edgelesssys/nunki/internal/coordapi"
	"github.com/edgelesssys/nunki/internal/grpc/dialer"
	"github.com/edgelesssys/nunki/internal/manifest"
	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/validate"
	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set a manifest",
		Long:  `Set a manifest.`,
		RunE:  runSet,
	}

	cmd.Flags().StringP("manifest", "m", "", "path to manifest (.json) file")
	cobra.MarkFlagRequired(cmd.Flags(), "manifest")
	cmd.Flags().StringP("coordinator", "c", "", "endpoint the coordinator can be reached at")
	cobra.MarkFlagRequired(cmd.Flags(), "coordinator")

	return cmd
}

func runSet(cmd *cobra.Command, args []string) error {
	flags, err := parseSetFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	logger, err := newCLILogger(cmd)
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

	paths, err := findGenerateTargets(args, logger)
	if err != nil {
		return fmt.Errorf("finding yaml files: %w", err)
	}

	policies, err := policiesFromKubeResources(paths)
	if err != nil {
		return fmt.Errorf("finding kube resources with policy: %w", err)
	}
	if err := checkPoliciesMatchManifest(policies, m.Policies); err != nil {
		return fmt.Errorf("checking policies match manifest: %w", err)
	}

	trustedIDKeyDigestHashes, err := m.ReferenceValues.SNP.TrustedIDKeyHashes.ByteSlices()
	if err != nil {
		return fmt.Errorf("failed to convert TrustedIDKeyHashes from manifest to byte slices: %w", err)
	}
	validateOptsGen := &snp.StaticValidateOptsGenerator{
		Opts: &validate.Options{
			GuestPolicy: abi.SnpPolicy{
				Debug: false,
				SMT:   true,
			},
			VMPL: new(int), // VMPL0
			MinimumTCB: kds.TCBParts{
				BlSpl:    m.ReferenceValues.SNP.MinimumTCB.BootloaderVersion.UInt8(),
				TeeSpl:   m.ReferenceValues.SNP.MinimumTCB.TEEVersion.UInt8(),
				SnpSpl:   m.ReferenceValues.SNP.MinimumTCB.SNPVersion.UInt8(),
				UcodeSpl: m.ReferenceValues.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
			},
			MinimumLaunchTCB: kds.TCBParts{
				BlSpl:    m.ReferenceValues.SNP.MinimumTCB.BootloaderVersion.UInt8(),
				TeeSpl:   m.ReferenceValues.SNP.MinimumTCB.TEEVersion.UInt8(),
				SnpSpl:   m.ReferenceValues.SNP.MinimumTCB.SNPVersion.UInt8(),
				UcodeSpl: m.ReferenceValues.SNP.MinimumTCB.MicrocodeVersion.UInt8(),
			},
			PermitProvisionalFirmware: true,
			TrustedIDKeyHashes:        trustedIDKeyDigestHashes,
			RequireIDBlock:            true,
		},
	}
	dialer := dialer.New(atls.NoIssuer, snp.NewValidator(validateOptsGen, logger), &net.Dialer{})

	conn, err := dialer.Dial(cmd.Context(), flags.coordinator)
	if err != nil {
		return fmt.Errorf("Error: failed to dial coordinator: %w", err)
	}
	defer conn.Close()

	client := coordapi.NewCoordAPIClient(conn)
	req := &coordapi.SetManifestRequest{
		Manifest: manifestBytes,
		Policies: policyMapToBytesList(policies),
	}
	resp, err := client.SetManifest(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to set manifest: %w", err)
	}

	logger.Info("Manifest set successfully")

	if err := os.WriteFile("coordinator-root.pem", resp.CACert, 0o644); err != nil {
		return fmt.Errorf("failed to write root certificate: %w", err)
	}
	if err := os.WriteFile("mesh-root.pem", resp.IntermCert, 0o644); err != nil {
		return fmt.Errorf("failed to write root certificate: %w", err)
	}

	return nil
}

type setFlags struct {
	manifestPath string
	coordinator  string
}

func parseSetFlags(cmd *cobra.Command) (*setFlags, error) {
	flags := &setFlags{}
	var err error

	flags.manifestPath, err = cmd.Flags().GetString("manifest")
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest flag: %w", err)
	}
	flags.coordinator, err = cmd.Flags().GetString("coordinator")
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinator flag: %w", err)
	}

	return flags, nil
}

func policyMapToBytesList(m map[string]deployment) [][]byte {
	var policies [][]byte
	for _, depl := range m {
		policies = append(policies, depl.policy)
	}
	return policies
}
