package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/edgelesssys/nunki/internal/atls"
	"github.com/edgelesssys/nunki/internal/attestation/snp"
	"github.com/edgelesssys/nunki/internal/coordapi"
	"github.com/edgelesssys/nunki/internal/fsstore"
	"github.com/edgelesssys/nunki/internal/grpc/dialer"
	"github.com/edgelesssys/nunki/internal/manifest"
	"github.com/edgelesssys/nunki/internal/spinner"
	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set [flags] paths...",
		Short: "Set the given manifest at the coordinator",
		Long: `
		Set the given manifest at the coordinator.

		This will connect to the given Coordinator using aTLS. During the connection
		initialization, the remote attestation of the Coordinator CVM happens and
		the connection will only be successful if the Coordinator conforms with the
		reference values embedded into the CLI.

		After the connection is established, the manifest is set. The Coordinator
		will re-generate the mesh root certificate and accept new workloads to
		issuer certificates.
		`,
		RunE: runSet,
	}

	cmd.Flags().StringP("manifest", "m", manifestFilename, "path to manifest (.json) file")
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

	kdsDir, err := cachedir("kds")
	if err != nil {
		return fmt.Errorf("getting cache dir: %w", err)
	}
	logger.Debug("Using KDS cache dir", "dir", kdsDir)

	validateOptsGen := newCoordinatorValidateOptsGen()
	kdsCache := fsstore.New(kdsDir, logger)
	kdsGetter := snp.NewCachedHTTPSGetter(kdsCache, snp.NeverCGTicker, logger)
	validator := snp.NewValidator(validateOptsGen, kdsGetter, logger)
	dialer := dialer.New(atls.NoIssuer, validator, &net.Dialer{})

	conn, err := dialer.Dial(cmd.Context(), flags.coordinator)
	if err != nil {
		return fmt.Errorf("failed to dial coordinator: %w", err)
	}
	defer conn.Close()

	client := coordapi.NewCoordAPIClient(conn)
	req := &coordapi.SetManifestRequest{
		Manifest: manifestBytes,
		Policies: policyMapToBytesList(policies),
	}
	resp, err := setLoop(cmd.Context(), client, cmd.OutOrStdout(), req)
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

func setLoop(
	ctx context.Context, client coordapi.CoordAPIClient, out io.Writer, req *coordapi.SetManifestRequest,
) (resp *coordapi.SetManifestResponse, retErr error) {
	spinner := spinner.New("  Waiting for coordinator ", 500*time.Millisecond, out)
	spinner.Start()
	defer func() {
		if retErr != nil {
			spinner.Stop("\r❌\n")
		} else {
			spinner.Stop("\x1b[2K\r✔️ Connected to coordinator\n")
		}
	}()

	var rpcErr error
	for attempts := 0; attempts < 30; attempts++ {
		resp, rpcErr = client.SetManifest(ctx, req)
		if rpcErr == nil {
			return resp, nil
		}
		timer := time.NewTimer(1 * time.Second)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, rpcErr
}
