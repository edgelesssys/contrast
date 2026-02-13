// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/edgelesssys/contrast/sdk"
	"github.com/spf13/afero"
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

	resp, err := getCoordinatorState(cmd.Context(), kdsDir, manifestBytes, flags.coordinator, log)
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

// getCoordinatorState calls GetManifests on the coordinator's userapi via aTLS.
func getCoordinatorState(ctx context.Context, kdsDir string, manifestBytes []byte, endpoint string, log *slog.Logger) (sdk.CoordinatorState, error) {
	var m manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return sdk.CoordinatorState{}, fmt.Errorf("unmarshalling manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return sdk.CoordinatorState{}, fmt.Errorf("validating manifest: %w", err)
	}

	kdsCache := fsstore.New(afero.NewBasePathFs(afero.NewOsFs(), kdsDir), log.WithGroup("kds-cache"))
	kdsGetter := certcache.NewCachedHTTPSGetter(kdsCache, certcache.NeverGCTicker, log.WithGroup("kds-getter"))
	validators, err := sdk.ValidatorsFromManifest(kdsGetter, &m, log)
	if err != nil {
		return sdk.CoordinatorState{}, fmt.Errorf("getting validators: %w", err)
	}
	dialer := dialer.New(atls.NoIssuer, validators, atls.NoMetrics, nil, log)

	log.Debug("Dialing coordinator", "endpoint", endpoint)

	conn, err := dialer.Dial(ctx, endpoint)
	if err != nil {
		return sdk.CoordinatorState{}, fmt.Errorf("dialing coordinator: %w", err)
	}
	defer conn.Close()

	log.Debug("Getting manifest")

	client := userapi.NewUserAPIClient(conn)
	resp, err := client.GetManifests(ctx, &userapi.GetManifestsRequest{})
	if err != nil {
		return sdk.CoordinatorState{}, fmt.Errorf("getting manifests: %w", err)
	}

	return sdk.CoordinatorState{
		Manifests: resp.Manifests,
		Policies:  resp.Policies,
		RootCA:    resp.RootCA,
		MeshCA:    resp.MeshCA,
	}, nil
}
