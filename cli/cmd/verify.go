package cmd

import (
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/snp"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/validate"
	"github.com/spf13/cobra"
)

// NewVerifyCmd creates the contrast verify subcommand.
func NewVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a contrast deployment",
		Long: `
		Verify a contrast deployment.

		This will connect to the given Coordinator using aTLS. During the connection
		initialization, the remote attestation of the Coordinator CVM happens and
		the connection will only be successful if the Coordinator conforms with the
		reference values embedded into the CLI.

		After the connection is established, the CLI will request the manifest history,
		all policies, and the certificates of the Coordinator certificate authority.
	`,
		RunE: runVerify,
	}

	// Override persistent workspace-dir flag with a default value.
	cmd.Flags().String("workspace-dir", verifyDir, "directory to write files to, if not set explicitly to another location")
	cmd.Flags().StringP("coordinator", "c", "", "endpoint the coordinator can be reached at")
	must(cobra.MarkFlagRequired(cmd.Flags(), "coordinator"))
	cmd.Flags().String("coordinator-policy-hash", DefaultCoordinatorPolicyHash, "override the expected policy hash of the coordinator")

	return cmd
}

func runVerify(cmd *cobra.Command, _ []string) error {
	flags, err := parseVerifyFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	log, err := newCLILogger(cmd)
	if err != nil {
		return err
	}
	log.Debug("Starting verification")

	kdsDir, err := cachedir("kds")
	if err != nil {
		return fmt.Errorf("getting cache dir: %w", err)
	}
	log.Debug("Using KDS cache dir", "dir", kdsDir)

	validateOptsGen := newCoordinatorValidateOptsGen(flags.policy)
	kdsCache := fsstore.New(kdsDir, log.WithGroup("kds-cache"))
	kdsGetter := snp.NewCachedHTTPSGetter(kdsCache, snp.NeverGCTicker, log.WithGroup("kds-getter"))
	validator := snp.NewValidator(validateOptsGen, kdsGetter, log.WithGroup("snp-validator"))
	dialer := dialer.New(atls.NoIssuer, validator, &net.Dialer{})

	log.Debug("Dialing coordinator", "endpoint", flags.coordinator)
	conn, err := dialer.Dial(cmd.Context(), flags.coordinator)
	if err != nil {
		return fmt.Errorf("Error: failed to dial coordinator: %w", err)
	}
	defer conn.Close()

	log.Debug("Getting manifest")
	client := userapi.NewUserAPIClient(conn)
	resp, err := client.GetManifests(cmd.Context(), &userapi.GetManifestsRequest{})
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}
	log.Debug("Got response")

	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Successfully verified coordinator")

	filelist := map[string][]byte{
		coordRootPEMFilename: resp.CoordinatorRoot,
		meshRootPEMFilename:  resp.MeshRoot,
	}
	for i, m := range resp.Manifests {
		filelist[fmt.Sprintf("manifest.%d.json", i)] = m
	}
	for _, p := range resp.Policies {
		sha256sum := sha256.Sum256(p)
		pHash := manifest.NewHexString(sha256sum[:])
		filelist[fmt.Sprintf("policy.%s.rego", pHash)] = p
	}
	if err := writeFilelist(flags.workspaceDir, filelist); err != nil {
		return fmt.Errorf("writing filelist: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✔️ Wrote Coordinator configuration and keys to %s\n", flags.workspaceDir)
	fmt.Fprintln(cmd.OutOrStdout(), "  Please verify the manifest history and policies")

	return nil
}

type verifyFlags struct {
	coordinator  string
	workspaceDir string
	policy       []byte
}

func parseVerifyFlags(cmd *cobra.Command) (*verifyFlags, error) {
	coordinator, err := cmd.Flags().GetString("coordinator")
	if err != nil {
		return nil, err
	}
	workspaceDir, err := cmd.Flags().GetString("workspace-dir")
	if err != nil {
		return nil, err
	}
	policy, err := decodeCoordinatorPolicyHash(cmd.Flags())
	if err != nil {
		return nil, err
	}

	return &verifyFlags{
		coordinator:  coordinator,
		workspaceDir: workspaceDir,
		policy:       policy,
	}, nil
}

func newCoordinatorValidateOptsGen(hostData []byte) *snp.StaticValidateOptsGenerator {
	defaultManifest := manifest.Default()
	trustedMeasurement, err := defaultManifest.ReferenceValues.SNP.TrustedMeasurement.Bytes()
	if err != nil {
		panic(err) // We are decoding known values, tests should catch any failure.
	}
	if trustedMeasurement == nil {
		// This is required to prevent an empty measurement in the manifest from disabling the measurement check.
		trustedMeasurement = make([]byte, 48)
	}

	return &snp.StaticValidateOptsGenerator{
		Opts: &validate.Options{
			HostData:    hostData,
			Measurement: trustedMeasurement,
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
		},
	}
}

func writeFilelist(dir string, filelist map[string][]byte) error {
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}
	for filename, contents := range filelist {
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, contents, 0o644); err != nil {
			return fmt.Errorf("writing %q: %w", path, err)
		}
	}
	return nil
}
