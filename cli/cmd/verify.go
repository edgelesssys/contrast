// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bytes"
	"cmp"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/cryptohelpers"
	"github.com/edgelesssys/contrast/internal/fsstore"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/history/configmapstore"
	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/edgelesssys/contrast/sdk"
	"github.com/edgelesssys/contrast/sdk/apiv1"
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
	cmd.Flags().String("coordinator-http", "", "base URL of the coordinator's HTTP API, for example http://192.0.2.1:1314, required if a minimum API version is pinned")
	must(cmd.Flags().MarkHidden("coordinator-http"))
	addCollateralProxyFlag(cmd)
	if insecureRuntimesAllowed() {
		cmd.Flags().Bool("INSECURE", false, "allow verification of insecure (non-CC) deployments")
	}
	cmd.Flags().String("minimum-api-version", "", "oldest Coordinator HTTP API version to accept, for example v1")
	must(cmd.Flags().MarkHidden("minimum-api-version"))

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

	var mnfst manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &mnfst); err != nil {
		return fmt.Errorf("unmarshalling manifest: %w", err)
	}
	if err := validateInsecureManifest(&mnfst, flags.allowInsecureRuntimes); err != nil {
		return err
	}

	kdsDir, err := cachedir("kds")
	if err != nil {
		return fmt.Errorf("getting cache dir: %w", err)
	}
	log.Debug("Using KDS cache dir", "dir", kdsDir)

	resp, err := getCoordinatorState(cmd.Context(), kdsDir, mnfst, manifestBytes, flags.coordinator, flags.coordinatorHTTP, flags.collateralProxyURL, flags.minimumAPIVersion, log)
	if err != nil {
		return fmt.Errorf("getting manifests: %w", err)
	}

	log.Debug("Got response")

	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Successfully verified Coordinator CVM based on reference values from manifest")

	filelist := map[string][]byte{
		coordRootPEMFilename:         resp.RootCA,
		meshCAPEMFilename:            resp.MeshCA,
		latestTransitionHashFilename: hex.AppendEncode(nil, resp.LatestTransitionHash),
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
	historyResources, err := configmapstore.RecoverConfigMaps(resp.Manifests, resp.Policies, resp.LatestTransitionHash, resp.LatestTransitionSignature)
	if err != nil {
		return fmt.Errorf("getting Coordinator history: %w", err)
	}
	historyBytes, err := kuberesource.EncodeResources(historyResources...)
	if err != nil {
		return fmt.Errorf("encoding Coordinator history: %w", err)
	}
	filelist[historyFilename] = historyBytes
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
	manifestPath          string
	coordinator           string
	coordinatorHTTP       string
	workspaceDir          string
	collateralProxyURL    string
	allowInsecureRuntimes bool
	minimumAPIVersion     string
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
	coordinatorHTTP, err := cmd.Flags().GetString("coordinator-http")
	if err != nil {
		return nil, err
	}
	workspaceDir, err := cmd.Flags().GetString("workspace-dir")
	if err != nil {
		return nil, err
	}
	collateralProxyURL, err := cmd.Flags().GetString("collateral-proxy")
	if err != nil {
		return nil, err
	}
	allowInsecureRuntimes := false
	if cmd.Flags().Lookup("INSECURE") != nil {
		allowInsecureRuntimes, err = cmd.Flags().GetBool("INSECURE")
		if err != nil {
			return nil, err
		}
	}
	minimumAPIVersion, err := cmd.Flags().GetString("minimum-api-version")
	if err != nil {
		return nil, err
	}
	if minimumAPIVersion != "" {
		if _, err := apitypes.ParseAPIVersion(minimumAPIVersion); err != nil {
			return nil, fmt.Errorf("invalid minimum-api-version: %w", err)
		}
	}

	if workspaceDir != "" {
		// Prepend default path with workspaceDir
		if !cmd.Flags().Changed("manifest") {
			manifestPath = filepath.Join(workspaceDir, manifestFilename)
		}
	}

	return &verifyFlags{
		manifestPath:          manifestPath,
		coordinator:           coordinator,
		coordinatorHTTP:       coordinatorHTTP,
		workspaceDir:          workspaceDir,
		collateralProxyURL:    collateralProxyURL,
		allowInsecureRuntimes: allowInsecureRuntimes,
		minimumAPIVersion:     minimumAPIVersion,
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

// getCoordinatorState fetches and verifies the state of the Coordinator running the expected manifest m.
//
// If the minimum-api-version flag or m pin a minimum API version, it uses the Coordinator's HTTP API.
// Otherwise, it calls GetManifests on the Coordinator's userapi via aTLS.
func getCoordinatorState(ctx context.Context, kdsDir string, m manifest.Manifest, manifestBytes []byte, endpoint, httpEndpoint, collateralProxy, minimumAPIVersion string, log *slog.Logger) (sdk.CoordinatorState, error) {
	if err := m.Validate(); err != nil {
		return sdk.CoordinatorState{}, fmt.Errorf("validating manifest: %w", err)
	}
	if pin := cmp.Or(minimumAPIVersion, m.MinimumAPIVersion); pin != "" {
		if httpEndpoint == "" {
			return sdk.CoordinatorState{}, fmt.Errorf("API version %s is pinned, but the coordinator-http flag isn't set", pin)
		}
		return getCoordinatorStateHTTP(ctx, kdsDir, httpEndpoint, manifestBytes, collateralProxy, pin, log)
	}

	kdsCache := fsstore.New(afero.NewBasePathFs(afero.NewOsFs(), kdsDir), log.WithGroup("kds-cache"))
	kdsGetter := certcache.NewCachedHTTPSGetter(kdsCache, certcache.NeverGCTicker, log.WithGroup("kds-getter"), collateralProxy)
	validator, err := m.CoordinatorValidator(log, kdsGetter)
	if err != nil {
		return sdk.CoordinatorState{}, fmt.Errorf("getting validators: %w", err)
	}
	dialer := dialer.New(atls.NoIssuer, validator, atls.NoMetrics, nil, log)

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
		Manifests:                 resp.Manifests,
		Policies:                  resp.Policies,
		RootCA:                    resp.RootCA,
		MeshCA:                    resp.MeshCA,
		LatestTransitionHash:      resp.LatestTransition.TransitionHash,
		LatestTransitionSignature: resp.LatestTransition.Signature,
	}, nil
}

// getCoordinatorStateHTTP fetches and verifies the Coordinator state via the Coordinator's HTTP API at baseURL.
//
// Unlike the aTLS handshake of the userapi, the attestation is validated against the reference values of the latest manifest
// the Coordinator reports.
func getCoordinatorStateHTTP(ctx context.Context, kdsDir, baseURL string, expectedManifest []byte, collateralProxy, minimumAPIVersion string, log *slog.Logger) (sdk.CoordinatorState, error) {
	client := sdk.New(baseURL).
		WithSlog(log).
		WithFSStore(afero.NewBasePathFs(afero.NewOsFs(), kdsDir)).
		WithCollateralProxy(collateralProxy).
		WithMinimumAPIVersion(minimumAPIVersion)

	version, err := client.NegotiateAPIVersion(ctx)
	if err != nil {
		return sdk.CoordinatorState{}, fmt.Errorf("negotiating API version: %w", err)
	}

	nonce, err := cryptohelpers.GenerateRandomBytes(cryptohelpers.RNGLengthDefault)
	if err != nil {
		return sdk.CoordinatorState{}, fmt.Errorf("generating nonce: %w", err)
	}

	log.Debug("Getting attestation", "url", baseURL, "apiVersion", version)

	var state sdk.CoordinatorState
	switch version {
	case apiv1.Version:
		attestation, err := client.V1().GetAttestation(ctx, nonce)
		if err != nil {
			return sdk.CoordinatorState{}, fmt.Errorf("getting attestation: %w", err)
		}
		v1State, err := client.V1().ValidateAttestation(ctx, nonce, attestation)
		if err != nil {
			return sdk.CoordinatorState{}, fmt.Errorf("validating attestation: %w", err)
		}
		state = sdk.CoordinatorState(*v1State)
	default:
		return sdk.CoordinatorState{}, fmt.Errorf("verifying isn't implemented for API version %q", version)
	}

	if len(state.Manifests) == 0 || !bytes.Equal(expectedManifest, state.Manifests[len(state.Manifests)-1]) {
		return sdk.CoordinatorState{}, errors.New("active manifest does not match expected manifest")
	}
	return state, nil
}
