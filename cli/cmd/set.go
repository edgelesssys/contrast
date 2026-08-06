// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/edgelesssys/contrast/apitypes"
	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/cryptohelpers"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	grpcRetry "github.com/edgelesssys/contrast/internal/grpc/retry"
	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/retry"
	"github.com/edgelesssys/contrast/internal/spinner"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/edgelesssys/contrast/sdk"
	"github.com/edgelesssys/contrast/sdk/apiv1"
	"github.com/spf13/afero"
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
	cmd.Flags().Bool("atomic", false, "only set the manifest if the coordinator's state matches the latest transition hash")
	cmd.Flags().String("latest-transition", "", "latest transition hash set at the coordinator (hex string)")
	cmd.Flags().StringP("signature", "s", "", "path to a detached transition signature (DER) file")
	must(cmd.MarkFlagFilename("signature"))
	cmd.Flags().Bool("experimental-http", false, "use the new HTTP API. Falls back to gRPC")
	cmd.Flags().String("experimental-http-url", "",
		"base URL of the coordinator's HTTP API, if it isn't reachable at the coordinator endpoint. Implies --experimental-http")
	cmd.Flags().String("experimental-http-version", "",
		"pin the HTTP API to this version instead of negotiating the newest one both sides support")
	addCollateralProxyFlag(cmd)

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
	var signatureBytes []byte
	if flags.signaturePath != "" {
		signatureBytes, err = os.ReadFile(flags.signaturePath)
		if err != nil {
			return fmt.Errorf("reading signature file: %w", err)
		}
	}

	paths, err := findYamlFiles(args)
	if err != nil {
		return fmt.Errorf("finding yaml files: %w", err)
	}

	fileMap, _, err := extractTargets(paths, io.Discard, log)
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

	var previousTransitionHash []byte
	if flags.atomic {
		if flags.latestTransition == "" {
			data, err := os.ReadFile(filepath.Join(flags.workspaceDir, verifyDir, latestTransitionHashFilename))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("reading previous transition hash: %w", err)
			} else if errors.Is(err, os.ErrNotExist) {
				data = []byte(strings.Repeat("00", history.HashSize)) // Assume initial set manifest
			}
			flags.latestTransition = string(data)
		}
		previousTransitionHash, err = hex.DecodeString(flags.latestTransition)
		if err != nil {
			return fmt.Errorf("decoding latest transition hash: %w", err)
		}
	}

	req := &userapi.SetManifestRequest{
		Manifest:               manifestBytes,
		Policies:               getInitdataDocuments(policies),
		PreviousTransitionHash: previousTransitionHash,
		Signature:              signatureBytes,
	}

	var resp *userapi.SetManifestResponse
	if flags.experimentalHTTP {
		resp, err = setViaHTTP(cmd.Context(), flags, &m, req, workloadOwnerKey, log)
		if errors.Is(err, errHTTPAPIUnavailable) {
			log.Info("Falling back to the gRPC API", "err", err)
			resp, err = nil, nil
		}
	}
	if resp == nil && err == nil {
		resp, err = setViaGRPC(cmd.Context(), flags, &m, req, workloadOwnerKey, cmd.OutOrStdout(), log)
	}
	if err != nil {
		grpcSt, ok := status.FromError(err)
		if ok {
			if grpcSt.Code() == codes.PermissionDenied {
				msg := "Permission denied."
				if signatureBytes != nil {
					msg += " Ensure the signature is valid and corresponds to the latest transition hash."
				} else if workloadOwnerKey == nil {
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

// errHTTPAPIUnavailable signals that the manifest could not be set over the HTTP API, and that the caller should fall back to gRPC.
var errHTTPAPIUnavailable = errors.New("coordinator HTTP API unavailable")

// setViaGRPC sets the manifest over the gRPC UserAPI, using aTLS.
func setViaGRPC(
	ctx context.Context, flags *setFlags, m *manifest.Manifest, req *userapi.SetManifestRequest,
	workloadOwnerKey *ecdsa.PrivateKey, out io.Writer, log *slog.Logger,
) (*userapi.SetManifestResponse, error) {
	kdsGetter, err := cachedHTTPSGetter(log, flags.collateralProxyURL)
	if err != nil {
		return nil, fmt.Errorf("configuring KDS cache: %w", err)
	}
	validator, err := m.CoordinatorValidator(log, kdsGetter)
	if err != nil {
		return nil, fmt.Errorf("getting validators: %w", err)
	}

	var dialr *dialer.Dialer
	if workloadOwnerKey == nil {
		dialr = dialer.New(atls.NoIssuer, validator, atls.NoMetrics, nil, log)
	} else {
		dialr = dialer.NewWithKey(atls.NoIssuer, validator, atls.NoMetrics, nil, workloadOwnerKey, log)
	}

	conn, err := dialr.Dial(ctx, flags.coordinator)
	if err != nil {
		return nil, fmt.Errorf("failed to dial coordinator: %w", err)
	}
	defer conn.Close()

	return setLoop(ctx, userapi.NewUserAPIClient(conn), out, req)
}

// setViaHTTP sets the manifest over the HTTP API, using the SDK.
//
// Unlike aTLS, the transport carries no attestation, so the Coordinator is attested explicitly
// before the manifest is sent, against the reference values of the manifest that's being set.
func setViaHTTP(
	ctx context.Context, flags *setFlags, m *manifest.Manifest, req *userapi.SetManifestRequest,
	workloadOwnerKey *ecdsa.PrivateKey, log *slog.Logger,
) (*userapi.SetManifestResponse, error) {
	baseURL := flags.experimentalHTTPURL
	if baseURL == "" {
		var err error
		baseURL, err = httpAPIBaseURL(flags.coordinator)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errHTTPAPIUnavailable, err)
		}
	}
	log.Debug("Using coordinator HTTP API", "url", baseURL)

	kdsDir, err := cachedir("kds")
	if err != nil {
		return nil, fmt.Errorf("getting cache dir: %w", err)
	}
	client := sdk.New().
		WithBaseURL(baseURL).
		WithSlog(log).
		WithCollateralProxy(flags.collateralProxyURL).
		WithFSStore(afero.NewBasePathFs(afero.NewOsFs(), kdsDir)).
		WithExpectedManifest(m)

	if flags.experimentalHTTPVersion != "" {
		client = client.WithAPIVersion(flags.experimentalHTTPVersion)
	}
	if _, err := client.NegotiateAPIVersion(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", errHTTPAPIUnavailable, err)
	}

	// Never send the manifest to a Coordinator we haven't attested.
	state, err := attestCoordinator(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("%w: attesting coordinator: %w", errHTTPAPIUnavailable, err)
	}

	if len(req.Signature) == 0 && workloadOwnerKey != nil {
		req.Signature, err = signTransition(req.Manifest, state.LatestTransitionHash, workloadOwnerKey)
		if err != nil {
			return nil, fmt.Errorf("signing transition: %w", err)
		}
	}

	apiReq := &apitypes.SetManifestRequest{
		Manifest:               req.Manifest,
		Policies:               req.Policies,
		PreviousTransitionHash: req.PreviousTransitionHash,
		Signature:              req.Signature,
	}

	var resp *apitypes.SetManifestResponse
	switch flags.experimentalHTTPVersion {
	case "":
		// Use whichever version the Coordinator and the SDK agree on.
		resp, err = client.SetManifest(ctx, apiReq)
	case apiv1.Version:
		resp, err = client.V1().SetManifest(ctx, apiReq)
	default:
		return nil, fmt.Errorf("unsupported API version %q", flags.experimentalHTTPVersion)
	}
	if err != nil {
		return nil, err
	}
	return setManifestResponseFromAPI(resp), nil
}

// httpAPIBaseURL derives the HTTP API's base URL from the coordinator endpoint, which addresses the gRPC API.
func httpAPIBaseURL(coordinator string) (string, error) {
	host, _, err := net.SplitHostPort(coordinator)
	if err != nil {
		// The endpoint may not carry a port.
		host = coordinator
	}
	if host == "" {
		return "", fmt.Errorf("no host in coordinator endpoint %q", coordinator)
	}
	return "http://" + net.JoinHostPort(host, apitypes.Port), nil
}

// attestCoordinator verifies the Coordinator's attestation and returns its state.
func attestCoordinator(ctx context.Context, client *sdk.Client) (*sdk.CoordinatorState, error) {
	nonce, err := cryptohelpers.GenerateRandomBytes(cryptohelpers.RNGLengthDefault)
	if err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	attestation, err := client.GetAttestation(ctx, nonce)
	if err != nil {
		return nil, fmt.Errorf("getting attestation: %w", err)
	}
	state, err := client.ValidateAttestation(ctx, nonce, attestation)
	if err != nil {
		return nil, fmt.Errorf("validating attestation: %w", err)
	}
	return state, nil
}

// signTransition signs the transition to the given manifest with the workload owner key.
//
// This is the same signature that `contrast sign` produces, and that the gRPC API's client
// certificate authentication substitutes for.
func signTransition(manifestBytes, previousTransitionHash []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	if len(previousTransitionHash) != history.HashSize {
		return nil, fmt.Errorf("invalid latest transition hash byte length: got %d, want %d", len(previousTransitionHash), history.HashSize)
	}
	tr := &history.Transition{
		ManifestHash:           history.Digest(manifestBytes),
		PreviousTransitionHash: [history.HashSize]byte(previousTransitionHash),
	}
	transitionHash := tr.Digest()
	signingHash := sha256.Sum256(hex.AppendEncode(nil, transitionHash[:]))
	return ecdsa.SignASN1(rand.Reader, key, signingHash[:])
}

// setManifestResponseFromAPI converts the HTTP API's response to its gRPC equivalent.
func setManifestResponseFromAPI(resp *apitypes.SetManifestResponse) *userapi.SetManifestResponse {
	out := &userapi.SetManifestResponse{
		RootCA: resp.RootCA,
		MeshCA: resp.MeshCA,
	}
	if resp.SeedSharesDoc == nil {
		return out
	}
	shares := make([]*userapi.SeedShare, 0, len(resp.SeedSharesDoc.SeedShares))
	for _, share := range resp.SeedSharesDoc.SeedShares {
		shares = append(shares, &userapi.SeedShare{
			PublicKey:     share.PublicKey,
			EncryptedSeed: share.EncryptedSeed,
		})
	}
	out.SeedSharesDoc = &userapi.SeedShareDocument{
		Salt:       resp.SeedSharesDoc.Salt,
		SeedShares: shares,
	}
	return out
}

type setFlags struct {
	manifestPath            string
	coordinator             string
	workloadOwnerKeyPath    string
	atomic                  bool
	latestTransition        string
	signaturePath           string
	workspaceDir            string
	collateralProxyURL      string
	experimentalHTTP        bool
	experimentalHTTPURL     string
	experimentalHTTPVersion string
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
	flags.atomic, err = cmd.Flags().GetBool("atomic")
	if err != nil {
		return nil, fmt.Errorf("getting atomic flag: %w", err)
	}
	flags.latestTransition, err = cmd.Flags().GetString("latest-transition")
	if err != nil {
		return nil, fmt.Errorf("getting latest-transition flag: %w", err)
	}
	if !flags.atomic && flags.latestTransition != "" {
		return nil, fmt.Errorf("\"latest-transition\" flag cannot be set without \"atomic\" flag")
	}
	flags.signaturePath, err = cmd.Flags().GetString("signature")
	if err != nil {
		return nil, fmt.Errorf("getting signature flag: %w", err)
	}
	flags.experimentalHTTP, err = cmd.Flags().GetBool("experimental-http")
	if err != nil {
		return nil, fmt.Errorf("getting experimental-http flag: %w", err)
	}
	flags.experimentalHTTPURL, err = cmd.Flags().GetString("experimental-http-url")
	if err != nil {
		return nil, fmt.Errorf("getting experimental-http-url flag: %w", err)
	}
	flags.experimentalHTTPVersion, err = cmd.Flags().GetString("experimental-http-version")
	if err != nil {
		return nil, fmt.Errorf("getting experimental-http-version flag: %w", err)
	}
	// Both flags configure the HTTP API, so either of them opts into it.
	if flags.experimentalHTTPURL != "" || flags.experimentalHTTPVersion != "" {
		flags.experimentalHTTP = true
	}
	flags.workspaceDir, err = cmd.Flags().GetString("workspace-dir")
	if err != nil {
		return nil, fmt.Errorf("getting workspace-dir flag: %w", err)
	}
	flags.collateralProxyURL, err = cmd.Flags().GetString("collateral-proxy")
	if err != nil {
		return nil, fmt.Errorf("getting collateral-proxy flag: %w", err)
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
		if len(manifst.WorkloadOwnerPubKeys) == 0 {
			log.Warn("No workload owner keys in manifest. Further manifest updates will be rejected by the Coordinator")
			return workloadOwnerKey, nil
		}
		log.Debug("Workload owner keys in manifest", "keys", manifst.WorkloadOwnerPubKeys)
		ownerKeyHex := manifest.MarshalWorkloadOwnerPubKey(&workloadOwnerKey.PublicKey)
		if !slices.Contains(manifst.WorkloadOwnerPubKeys, ownerKeyHex) {
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
