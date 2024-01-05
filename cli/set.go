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
	must(cobra.MarkFlagRequired(cmd.Flags(), "manifest"))
	cmd.Flags().StringP("coordinator", "c", "", "endpoint the coordinator can be reached at")
	must(cobra.MarkFlagRequired(cmd.Flags(), "coordinator"))

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

	validateOptsGen := newCoordinatorValidateOptsGen()

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

	fmt.Fprintln(cmd.OutOrStdout(), "Manifest set successfully")

	filelist := map[string][]byte{
		coordRootPEMFilename:   resp.CACert,
		coordIntermPEMFilename: resp.IntermCert,
	}
	if err := writeFilelist(".", filelist); err != nil {
		return fmt.Errorf("writing filelist: %w", err)
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
