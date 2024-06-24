// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"crypto/sha256"
	"encoding/json"
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
	"github.com/spf13/cobra"
)

// NewVerifyCmd creates the contrast verify subcommand.
func NewVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a contrast deployment",
		Long: `Verify a contrast deployment.

This will connect to the given Coordinator using aTLS. During the connection
initialization, the remote attestation of the Coordinator CVM happens and
the connection will only be successful if the Coordinator conforms with the
reference values embedded into the CLI.

After the connection is established, the CLI will request the manifest history,
all policies, and the certificates of the Coordinator certificate authority.`,
		RunE: withTelemetry(runVerify),
	}

	cmd.Flags().StringP("manifest", "m", manifestFilename, "path to manifest (.json) file")
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
		coordRootPEMFilename: resp.RootCA,
		meshCAPEMFilename:    resp.MeshCA,
	}
	for i, m := range resp.Manifests {
		filelist[fmt.Sprintf("manifest.%d.json", i)] = m
	}
	for _, p := range resp.Policies {
		sha256sum := sha256.Sum256(p)
		pHash := manifest.NewHexString(sha256sum[:])
		filelist[fmt.Sprintf("policy.%s.rego", pHash)] = p
	}
	if err := writeFilelist(filepath.Join(flags.workspaceDir, verifyDir), filelist); err != nil {
		return fmt.Errorf("writing filelist: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✔️ Wrote Coordinator configuration and keys to %s\n", filepath.Join(flags.workspaceDir, verifyDir))
	fmt.Fprintln(cmd.OutOrStdout(), "  Please verify the manifest history and policies")

	return nil
}

type verifyFlags struct {
	manifestPath string
	coordinator  string
	workspaceDir string
	policy       []byte
}

func parseVerifyFlags(cmd *cobra.Command) (*verifyFlags, error) {
	manifestPath, err := cmd.Flags().GetString("manifest")
	if err != nil {
		return nil, err
	}
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

	if workspaceDir != "" {
		// Prepend default path with workspaceDir
		if !cmd.Flags().Changed("manifest") {
			manifestPath = filepath.Join(workspaceDir, manifestFilename)
		}
	}

	return &verifyFlags{
		manifestPath: manifestPath,
		coordinator:  coordinator,
		workspaceDir: workspaceDir,
		policy:       policy,
	}, nil
}

func newCoordinatorValidateOptsGen(mnfst manifest.Manifest, hostData []byte) (*snp.StaticValidateOptsGenerator, error) {
	validateOpts, err := mnfst.SNPValidateOpts()
	if err != nil {
		return nil, err
	}
	validateOpts.HostData = hostData
	return &snp.StaticValidateOptsGenerator{
		Opts: validateOpts,
	}, nil
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
