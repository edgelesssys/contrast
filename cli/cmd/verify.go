// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/sdk"
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
	cmd.SetOut(commandOut())

	cmd.Flags().StringP("manifest", "m", manifestFilename, "path to manifest (.json) file")
	cmd.Flags().StringP("coordinator", "c", "", "endpoint the coordinator can be reached at")
	must(cobra.MarkFlagRequired(cmd.Flags(), "coordinator"))

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

	kdsDir, err := cachedir("kds")
	if err != nil {
		return fmt.Errorf("getting cache dir: %w", err)
	}
	log.Debug("Using KDS cache dir", "dir", kdsDir)

	sdkClient := sdk.NewWithSlog(log)
	//nolint:staticcheck // We'll move this function here after removing it from the SDK.
	resp, err := sdkClient.GetCoordinatorState(cmd.Context(), kdsDir, manifestBytes, flags.coordinator)
	if err != nil {
		return fmt.Errorf("getting manifests: %w", err)
	}

	log.Debug("Got response")

	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Successfully verified Coordinator CVM based on reference values from manifest")

	filelist := map[string][]byte{
		coordRootPEMFilename: resp.RootCA,
		meshCAPEMFilename:    resp.MeshCA,
	}
	for i, m := range resp.Manifests {
		filelist[fmt.Sprintf("manifest.%d.json", i)] = m
	}
	for _, p := range resp.Policies {
		initdata := initdata.Raw(p)
		digest, err := initdata.Digest()
		if err != nil {
			return fmt.Errorf("calculating initdata digest: %w", err)
		}
		filelist[fmt.Sprintf("initdata.%x.toml", digest)] = initdata
	}
	if err := writeFilelist(filepath.Join(flags.workspaceDir, verifyDir), filelist); err != nil {
		return fmt.Errorf("writing filelist: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✔️ Wrote Coordinator configuration and keys to %s\n", filepath.Join(flags.workspaceDir, verifyDir))

	// Check that the current manifest is the expected one.
	if len(resp.Manifests) < 1 {
		return fmt.Errorf("failed to verify Coordinator manifest: manifest history is empty")
	}
	if !bytes.Equal(manifestBytes, resp.Manifests[len(resp.Manifests)-1]) {
		return fmt.Errorf("failed to verify Coordinator manifest: active manifest does not match expected manifest")
	}

	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Manifest active at Coordinator matches expected manifest")
	fmt.Fprintln(cmd.OutOrStdout(), "  Please verify the manifest history and policies")

	return nil
}

type verifyFlags struct {
	manifestPath string
	coordinator  string
	workspaceDir string
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
