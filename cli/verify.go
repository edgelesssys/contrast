package main

import (
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"path/filepath"

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

func newVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a nunki deployment",
		Long:  `Verify a manifest.`,
		RunE:  runVerify,
	}

	cmd.Flags().StringP("output", "o", verifyDir, "directory to write files to")
	cmd.Flags().StringP("coordinator", "c", "", "endpoint the coordinator can be reached at")
	must(cobra.MarkFlagRequired(cmd.Flags(), "coordinator"))

	return cmd
}

func runVerify(cmd *cobra.Command, _ []string) error {
	flags, err := parseVerifyFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	logger, err := newCLILogger(cmd)
	if err != nil {
		return err
	}

	validateOptsGen := newCoordinatorValidateOptsGen()
	dialer := dialer.New(atls.NoIssuer, snp.NewValidator(validateOptsGen, logger), &net.Dialer{})

	conn, err := dialer.Dial(cmd.Context(), flags.coordinator)
	if err != nil {
		return fmt.Errorf("Error: failed to dial coordinator: %w", err)
	}
	defer conn.Close()

	client := coordapi.NewCoordAPIClient(conn)
	resp, err := client.GetManifests(cmd.Context(), &coordapi.GetManifestsRequest{})
	if err != nil {
		return fmt.Errorf("failed to set manifest: %w", err)
	}

	filelist := map[string][]byte{
		coordRootPEMFilename:   resp.CACert,
		coordIntermPEMFilename: resp.IntermCert,
	}
	for i, m := range resp.Manifests {
		filelist[fmt.Sprintf("manifest.%d.json", i)] = m
	}
	for _, p := range resp.Policies {
		sha256sum := sha256.Sum256(p)
		pHash := manifest.NewHexString(sha256sum[:])
		filelist[fmt.Sprintf("policy.%s.rego", pHash)] = p
	}
	if err := writeFilelist(flags.outputDir, filelist); err != nil {
		return fmt.Errorf("writing filelist: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Successfully verified coordinator")

	return nil
}

type verifyFlags struct {
	coordinator string
	outputDir   string
}

func parseVerifyFlags(cmd *cobra.Command) (*verifyFlags, error) {
	coordinator, err := cmd.Flags().GetString("coordinator")
	if err != nil {
		return nil, err
	}
	outputDir, err := cmd.Flags().GetString("output")
	if err != nil {
		return nil, err
	}

	return &verifyFlags{
		coordinator: coordinator,
		outputDir:   outputDir,
	}, nil
}

func newCoordinatorValidateOptsGen() *snp.StaticValidateOptsGenerator {
	trustedIDKeyDigests, err := (&manifest.HexStrings{
		"b2bcf1b11d9fb3f2e4e7979546844d26c30255fff0775f3af56f8295f361a7d1a34a54516d41abfff7320763a5b701d8",
		"22087e0b99b911c9cffccfd9550a054531c105d46ed6d31f948eae56bd2defa4887e2fc4207768ec610aa232ac7490c4",
	}).ByteSlices()
	if err != nil {
		panic(err) // We are decoding known values, tests should catch any failure.
	}

	return &snp.StaticValidateOptsGenerator{
		Opts: &validate.Options{
			GuestPolicy: abi.SnpPolicy{
				Debug: false,
				SMT:   true,
			},
			VMPL: new(int), // VMPL0
			MinimumTCB: kds.TCBParts{
				BlSpl:    3,
				TeeSpl:   0,
				SnpSpl:   8,
				UcodeSpl: 115,
			},
			MinimumLaunchTCB: kds.TCBParts{
				BlSpl:    3,
				TeeSpl:   0,
				SnpSpl:   8,
				UcodeSpl: 115,
			},
			PermitProvisionalFirmware: true,
			TrustedIDKeyHashes:        trustedIDKeyDigests,
			RequireIDBlock:            true,
		},
	}
}

func writeFilelist(dir string, filelist map[string][]byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	for filename, contents := range filelist {
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, contents, 0o644); err != nil {
			return fmt.Errorf("writing %q: %w", path, err)
		}
	}
	return nil
}
