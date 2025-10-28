// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	grpcRetry "github.com/edgelesssys/contrast/internal/grpc/retry"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/retry"
	"github.com/edgelesssys/contrast/internal/spinner"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/edgelesssys/contrast/sdk"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewSetCmd creates the contrast set subcommand.
func NewSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set [flags] paths...",
		Short: "Set the given manifest at the coordinator",
		Long: `Set the given manifest at the coordinator.

This will connect to the given Coordinator using aTLS. During the connection
initialization, the remote attestation of the Coordinator CVM happens and
the connection will only be successful if the Coordinator conforms with the
reference values embedded into the CLI.

After the connection is established, the manifest is set. The Coordinator
will re-generate the mesh CA certificate and accept new workloads to
issuer certificates.`,
		RunE: withTelemetry(runSet),
	}
	cmd.SetOut(commandOut())

	cmd.Flags().StringP("manifest", "m", manifestFilename, "path to manifest (.json) file")
	cmd.Flags().StringP("coordinator", "c", "", "endpoint the coordinator can be reached at")
	must(cobra.MarkFlagRequired(cmd.Flags(), "coordinator"))
	cmd.Flags().String("workload-owner-key", workloadOwnerPEM, "path to workload owner key (.pem) file")

	return cmd
}

func runSet(cmd *cobra.Command, args []string) error {
	flags, err := parseSetFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	log, err := newCLILogger(cmd)
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
	if err := m.Validate(); err != nil {
		return fmt.Errorf("validating manifest: %w", err)
	}

	workloadOwnerKey, err := loadWorkloadOwnerKey(flags.workloadOwnerKeyPath, &m, log)
	if errors.Is(err, os.ErrNotExist) {
		workloadOwnerKey = nil
	} else if err != nil {
		return fmt.Errorf("loading workload owner key: %w", err)
	}

	paths, err := findYamlFiles(args)
	if err != nil {
		return fmt.Errorf("finding yaml files: %w", err)
	}

	fileMap, err := extractTargets(paths, io.Discard, log)
	if err != nil {
		return fmt.Errorf("extracting targets from yaml files: %w", err)
	}
	policies, err := policiesFromKubeResources(fileMap)
	if err != nil {
		return fmt.Errorf("finding kube resources with policy: %w", err)
	}
	if err := checkPoliciesMatchManifest(policies, m.Policies); err != nil {
		return fmt.Errorf("checking policies match manifest: %w", err)
	}

	kdsGetter, err := cachedHTTPSGetter(log)
	if err != nil {
		return fmt.Errorf("configuring KDS cache: %w", err)
	}
	validators, err := sdk.ValidatorsFromManifest(kdsGetter, &m, log)
	if err != nil {
		return fmt.Errorf("getting validators: %w", err)
	}

	var dialr *dialer.Dialer
	if workloadOwnerKey == nil {
		dialr = dialer.New(atls.NoIssuer, validators, atls.NoMetrics, nil, log)
	} else {
		dialr = dialer.NewWithKey(atls.NoIssuer, validators, atls.NoMetrics, nil, workloadOwnerKey, log)
	}

	conn, err := dialr.Dial(cmd.Context(), flags.coordinator)
	if err != nil {
		return fmt.Errorf("failed to dial coordinator: %w", err)
	}
	defer conn.Close()

	client := userapi.NewUserAPIClient(conn)
	req := &userapi.SetManifestRequest{
		Manifest: manifestBytes,
		Policies: getInitdataDocuments(policies),
	}
	resp, err := setLoop(cmd.Context(), client, cmd.OutOrStdout(), req)
	if err != nil {
		grpcSt, ok := status.FromError(err)
		if ok {
			if grpcSt.Code() == codes.PermissionDenied {
				msg := "Permission denied."
				if workloadOwnerKey == nil {
					msg += " Specify a workload owner key with --workload-owner-key."
				} else {
					msg += " Ensure you are using a trusted workload owner key."
				}
				fmt.Fprintln(cmd.OutOrStdout(), msg)
			}
		}
		additionalHelp := ""
		if strings.Contains(err.Error(), "quote field MR_CONFIG_ID") || strings.Contains(err.Error(), "report field HOST_DATA") {
			additionalHelp = " (coordinator did not match the expectations, is the version correct and did you run `contrast generate`?)"
		}
		return fmt.Errorf("setting manifest%s: %w", additionalHelp, err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Manifest set successfully")

	filelist := map[string][]byte{
		coordRootPEMFilename: resp.RootCA,
		meshCAPEMFilename:    resp.MeshCA,
	}

	if resp.SeedSharesDoc != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "✔️ Seed shares received")
		seedShareFile, err := json.MarshalIndent(resp.SeedSharesDoc, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling seed shares: %w", err)
		}
		filelist[seedSharesFilename] = seedShareFile
	}

	if err := writeFilelist(flags.workspaceDir, filelist); err != nil {
		return fmt.Errorf("writing filelist: %w", err)
	}

	return nil
}

type setFlags struct {
	manifestPath         string
	coordinator          string
	workloadOwnerKeyPath string
	workspaceDir         string
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
	flags.workloadOwnerKeyPath, err = cmd.Flags().GetString("workload-owner-key")
	if err != nil {
		return nil, fmt.Errorf("getting workload-owner-key flag: %w", err)
	}
	flags.workspaceDir, err = cmd.Flags().GetString("workspace-dir")
	if err != nil {
		return nil, fmt.Errorf("getting workspace-dir flag: %w", err)
	}

	if flags.workspaceDir != "" {
		// Prepend default paths with workspaceDir
		if !cmd.Flags().Changed("manifest") {
			flags.manifestPath = path.Join(flags.workspaceDir, flags.manifestPath)
		}
		if !cmd.Flags().Changed("workload-owner-key") {
			flags.workloadOwnerKeyPath = path.Join(flags.workspaceDir, flags.workloadOwnerKeyPath)
		}
	}

	return flags, nil
}

func getInitdataDocuments(m []deployment) [][]byte {
	var initdataDocs [][]byte
	for _, depl := range m {
		initdataDocs = append(initdataDocs, depl.initdata)
	}
	return initdataDocs
}

func loadWorkloadOwnerKey(path string, manifst *manifest.Manifest, log *slog.Logger) (*ecdsa.PrivateKey, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading workload owner key: %w", err)
	}
	workloadOwnerKey, err := manifest.ParseWorkloadOwnerPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parsing workload owner key: %w", err)
	}

	// Check workload owner key configuration in manifest on set.
	if manifst != nil {
		if len(manifst.WorkloadOwnerKeyDigests) == 0 {
			log.Warn("No workload owner keys in manifest. Further manifest updates will be rejected by the Coordinator")
			return workloadOwnerKey, nil
		}
		log.Debug("Workload owner keys in manifest", "keys", manifst.WorkloadOwnerKeyDigests)
		ownerKeyHex := manifest.HashWorkloadOwnerKey(&workloadOwnerKey.PublicKey)
		if !slices.Contains(manifst.WorkloadOwnerKeyDigests, ownerKeyHex) {
			log.Warn("Workload owner key not found in manifest. This may lock you out from further updates")
		}
	}

	return workloadOwnerKey, nil
}

type setDoer struct {
	client userapi.UserAPIClient
	req    *userapi.SetManifestRequest

	resp *userapi.SetManifestResponse
}

func (d *setDoer) Do(ctx context.Context) error {
	resp, err := d.client.SetManifest(ctx, d.req)
	if err == nil {
		d.resp = resp
		return nil
	}
	return err
}

func setLoop(
	ctx context.Context, client userapi.UserAPIClient, out io.Writer, req *userapi.SetManifestRequest,
) (resp *userapi.SetManifestResponse, retErr error) {
	spinner := spinner.New("  Waiting for coordinator ", 500*time.Millisecond, out)
	spinner.Start()
	defer func() {
		if retErr != nil {
			spinner.Stop("\r❌\n")
		} else {
			spinner.Stop("\x1b[2K\r✔️ Connected to coordinator\n")
		}
	}()

	doer := &setDoer{
		client: client,
		req:    req,
	}

	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	retrier := retry.NewIntervalRetrier(doer, time.Second, grpcRetry.ServiceIsUnavailable)
	if err := retrier.Do(ctx); err != nil {
		return nil, err
	}

	return doer.resp, nil
}
