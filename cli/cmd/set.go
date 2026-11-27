// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bytes"
	"cmp"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/edgelesssys/contrast/apitypes"
	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
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
	cmd.Flags().StringP("coordinator", "c", "", "endpoint the coordinator can be reached at, required unless a minimum API version is pinned")
	cmd.Flags().String("workload-owner-key", workloadOwnerPEM, "path to workload owner key (.pem) file")
	cmd.Flags().Bool("atomic", false, "only set the manifest if the coordinator's state matches the latest transition hash")
	cmd.Flags().String("latest-transition", "", "latest transition hash set at the coordinator (hex string)")
	cmd.Flags().StringP("signature", "s", "", "path to a detached transition signature (DER) file")
	must(cmd.MarkFlagFilename("signature"))
	addCollateralProxyFlag(cmd)
	cmd.Flags().String("coordinator-http", "", "base URL of the coordinator's HTTP API, for example http://192.0.2.1:1314, required if a minimum API version is pinned")
	must(cmd.Flags().MarkHidden("coordinator-http"))
	cmd.Flags().String("minimum-api-version", "", "oldest Coordinator HTTP API version to accept, for example v1")
	must(cmd.Flags().MarkHidden("minimum-api-version"))

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

	policyDocs := getInitdataDocuments(policies)
	out := cmd.OutOrStdout()

	var rootCA, meshCA, seedSharesFile []byte
	if pin := cmp.Or(flags.minimumAPIVersion, m.MinimumAPIVersion); pin != "" {
		if flags.coordinatorHTTP == "" {
			return fmt.Errorf("API version %s is pinned, but the coordinator-http flag isn't set", pin)
		}
		result, err := setManifestHTTP(cmd.Context(), out, &httpSetRequest{
			baseURL:                flags.coordinatorHTTP,
			collateralProxy:        flags.collateralProxyURL,
			minimumAPIVersion:      pin,
			manifest:               manifestBytes,
			policies:               policyDocs,
			atomic:                 flags.atomic,
			previousTransitionHash: previousTransitionHash,
			signature:              signatureBytes,
			workloadOwnerKey:       workloadOwnerKey,
		}, log)
		if err != nil {
			var apiErr *apitypes.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
				printPermissionDeniedHelp(out, signatureBytes, workloadOwnerKey)
			}
			return setManifestError(err)
		}
		rootCA, meshCA = result.RootCA, result.MeshCA
		if result.SeedSharesDoc != nil {
			var indented bytes.Buffer
			if err := json.Indent(&indented, result.SeedSharesDoc, "", "  "); err != nil {
				return fmt.Errorf("formatting seed shares: %w", err)
			}
			seedSharesFile = indented.Bytes()
		}
	} else {
		if flags.coordinator == "" {
			return errors.New("required flag \"coordinator\" not set")
		}
		kdsGetter, err := cachedHTTPSGetter(log, flags.collateralProxyURL)
		if err != nil {
			return fmt.Errorf("configuring KDS cache: %w", err)
		}
		validator, err := m.CoordinatorValidator(log, kdsGetter)
		if err != nil {
			return fmt.Errorf("getting validators: %w", err)
		}

		var dialr *dialer.Dialer
		if workloadOwnerKey == nil {
			dialr = dialer.New(atls.NoIssuer, validator, atls.NoMetrics, nil, log)
		} else {
			dialr = dialer.NewWithKey(atls.NoIssuer, validator, atls.NoMetrics, nil, workloadOwnerKey, log)
		}

		conn, err := dialr.Dial(cmd.Context(), flags.coordinator)
		if err != nil {
			return fmt.Errorf("failed to dial coordinator: %w", err)
		}
		defer conn.Close()

		client := userapi.NewUserAPIClient(conn)
		req := &userapi.SetManifestRequest{
			Manifest:               manifestBytes,
			Policies:               policyDocs,
			PreviousTransitionHash: previousTransitionHash,
			Signature:              signatureBytes,
		}
		resp, err := setLoop(cmd.Context(), client, out, req)
		if err != nil {
			if status.Code(err) == codes.PermissionDenied {
				printPermissionDeniedHelp(out, signatureBytes, workloadOwnerKey)
			}
			return setManifestError(err)
		}
		rootCA, meshCA = resp.RootCA, resp.MeshCA
		if resp.SeedSharesDoc != nil {
			seedSharesFile, err = json.MarshalIndent(resp.SeedSharesDoc, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling seed shares: %w", err)
			}
		}
	}

	fmt.Fprintln(out, "✔️ Manifest set successfully")

	filelist := map[string][]byte{
		coordRootPEMFilename: rootCA,
		meshCAPEMFilename:    meshCA,
	}
	if seedSharesFile != nil {
		fmt.Fprintln(out, "✔️ Seed shares received")
		filelist[seedSharesFilename] = seedSharesFile
	}

	if err := writeFilelist(flags.workspaceDir, filelist); err != nil {
		return fmt.Errorf("writing filelist: %w", err)
	}

	return nil
}

// printPermissionDeniedHelp tells the user how to authorize a manifest update the Coordinator rejected.
func printPermissionDeniedHelp(out io.Writer, signature []byte, workloadOwnerKey *ecdsa.PrivateKey) {
	msg := "Permission denied."
	if signature != nil {
		msg += " Ensure the signature is valid and corresponds to the latest transition hash."
	} else if workloadOwnerKey == nil {
		msg += " Specify a workload owner key with --workload-owner-key."
	} else {
		msg += " Ensure you are using a trusted workload owner key."
	}
	fmt.Fprintln(out, msg)
}

func setManifestError(err error) error {
	additionalHelp := ""
	if strings.Contains(err.Error(), "quote field MR_CONFIG_ID") || strings.Contains(err.Error(), "report field HOST_DATA") {
		additionalHelp = " (coordinator did not match the expectations, is the version correct and did you run `contrast generate`?)"
	}
	return fmt.Errorf("setting manifest%s: %w", additionalHelp, err)
}

type setFlags struct {
	manifestPath         string
	coordinator          string
	workloadOwnerKeyPath string
	atomic               bool
	latestTransition     string
	signaturePath        string
	workspaceDir         string
	collateralProxyURL   string
	coordinatorHTTP      string
	minimumAPIVersion    string
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
	flags.workspaceDir, err = cmd.Flags().GetString("workspace-dir")
	if err != nil {
		return nil, fmt.Errorf("getting workspace-dir flag: %w", err)
	}
	flags.collateralProxyURL, err = cmd.Flags().GetString("collateral-proxy")
	if err != nil {
		return nil, fmt.Errorf("getting collateral-proxy flag: %w", err)
	}
	flags.coordinatorHTTP, err = cmd.Flags().GetString("coordinator-http")
	if err != nil {
		return nil, fmt.Errorf("getting coordinator-http flag: %w", err)
	}
	flags.minimumAPIVersion, err = cmd.Flags().GetString("minimum-api-version")
	if err != nil {
		return nil, fmt.Errorf("getting minimum-api-version flag: %w", err)
	}
	if flags.minimumAPIVersion != "" {
		if _, err := apitypes.ParseAPIVersion(flags.minimumAPIVersion); err != nil {
			return nil, fmt.Errorf("invalid minimum-api-version: %w", err)
		}
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

	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	if req.PreviousTransitionHash == nil {
		fetch := retry.DoerFunc(func(ctx context.Context) error {
			latest, err := getLatestTransitionGRPC(ctx, client)
			if err != nil {
				return err
			}
			req.PreviousTransitionHash = latest
			return nil
		})
		if err := retry.NewIntervalRetrier(fetch, time.Second, grpcRetry.ServiceIsUnavailable).Do(ctx); err != nil {
			return nil, fmt.Errorf("getting latest transition: %w", err)
		}
	}

	doer := &setDoer{
		client: client,
		req:    req,
	}

	// The request is identical in every attempt, meaning the Coordinator answers a repetition with the original response.
	retrier := retry.NewIntervalRetrier(doer, time.Second, grpcRetry.ServiceIsUnavailable)
	if err := retrier.Do(ctx); err != nil {
		return nil, err
	}

	return doer.resp, nil
}

// getLatestTransitionGRPC returns the hash of the Coordinator's latest transition, or all zeroes if it has no manifest yet.
func getLatestTransitionGRPC(ctx context.Context, client userapi.UserAPIClient) ([]byte, error) {
	resp, err := client.GetManifests(ctx, &userapi.GetManifestsRequest{})
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.FailedPrecondition && s.Message() == userapi.ErrNoManifest.Error() {
			return make([]byte, history.HashSize), nil
		}
		return nil, err
	}
	if latest := resp.GetLatestTransition().GetTransitionHash(); len(latest) == history.HashSize {
		return latest, nil
	}
	if len(resp.GetManifests()) == 0 {
		return nil, errors.New("coordinator reported neither its latest transition nor any manifests")
	}
	transitions := history.BuildTransitionChain(resp.GetManifests())
	latest := transitions[len(transitions)-1].Digest()
	return latest[:], nil
}

// httpSetRequest holds the inputs of setManifestHTTP.
type httpSetRequest struct {
	// baseURL is the root of the Coordinator's HTTP API.
	baseURL           string
	collateralProxy   string
	minimumAPIVersion string

	manifest []byte
	policies [][]byte
	// atomic is set if the manifest must only be set at previousTransitionHash.
	atomic                 bool
	previousTransitionHash []byte
	// signature is a detached signature of the transition. If it's nil, the transition is signed with workloadOwnerKey, if set.
	signature        []byte
	workloadOwnerKey *ecdsa.PrivateKey
}

// setManifestHTTP sets the manifest through the Coordinator's HTTP API and validates the attested result.
func setManifestHTTP(ctx context.Context, out io.Writer, req *httpSetRequest, log *slog.Logger) (result *apiv1.SetManifestResult, retErr error) {
	kdsDir, err := cachedir("kds")
	if err != nil {
		return nil, fmt.Errorf("getting cache dir: %w", err)
	}
	client := sdk.New(req.baseURL).
		WithSlog(log).
		WithFSStore(afero.NewBasePathFs(afero.NewOsFs(), kdsDir)).
		WithCollateralProxy(req.collateralProxy).
		WithMinimumAPIVersion(req.minimumAPIVersion)

	spinner := spinner.New("  Waiting for coordinator ", 500*time.Millisecond, out)
	spinner.Start()
	defer func() {
		if retErr != nil {
			spinner.Stop("\r❌\n")
		} else {
			spinner.Stop("\x1b[2K\r✔️ Connected to coordinator\n")
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	var send func(context.Context) (*apiv1.SetManifestResult, error)
	prepare := retry.DoerFunc(func(ctx context.Context) error {
		version, err := client.NegotiateAPIVersion(ctx)
		if err != nil {
			return fmt.Errorf("negotiating API version: %w", err)
		}
		switch version {
		case apiv1.Version:
			send, err = prepareSetManifestV1(ctx, client.V1(), req)
			return err
		default:
			return fmt.Errorf("setting a manifest isn't implemented for API version %q", version)
		}
	})
	// The preparation sends nothing that changes the Coordinator's state, so it's retried if the Coordinator can't be reached.
	if err := retry.NewIntervalRetrier(prepare, time.Second, isDialError).Do(ctx); err != nil {
		return nil, err
	}

	// The prepared request is repeated as is, which the Coordinator answers with its original response if the
	// first one was lost after the manifest was set.
	doer := retry.DoerFunc(func(ctx context.Context) error {
		var err error
		result, err = send(ctx)
		return err
	})
	if err := retry.NewIntervalRetrier(doer, time.Second, isRetriableSetError).Do(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

// prepareSetManifestV1 builds the request that sets the manifest through version v1 of the Coordinator's HTTP API, and returns a func that sends it.
func prepareSetManifestV1(ctx context.Context, api *apiv1.API, req *httpSetRequest) (func(context.Context) (*apiv1.SetManifestResult, error), error) {
	previousTransitionHash := req.previousTransitionHash
	if !req.atomic {
		var err error
		previousTransitionHash, err = getLatestTransitionV1(ctx, api)
		if err != nil {
			return nil, fmt.Errorf("getting latest transition: %w", err)
		}
	}

	signature := req.signature
	if signature == nil && req.workloadOwnerKey != nil {
		sign, err := mayAutoSign(req, previousTransitionHash)
		if err != nil {
			return nil, err
		}
		if sign {
			signature, err = sdk.SignManifestTransition(req.workloadOwnerKey, req.manifest, previousTransitionHash)
			if err != nil {
				return nil, fmt.Errorf("signing transition: %w", err)
			}
		}
	}

	return func(ctx context.Context) (*apiv1.SetManifestResult, error) {
		nonce, err := cryptohelpers.GenerateRandomBytes(apitypesv1.NonceSize)
		if err != nil {
			return nil, fmt.Errorf("generating nonce: %w", err)
		}
		return api.SetManifest(ctx, &apitypesv1.SetManifestRequest{
			Nonce:                  nonce,
			Manifest:               req.manifest,
			Policies:               req.policies,
			PreviousTransitionHash: previousTransitionHash,
			Signature:              signature,
		})
	}, nil
}

// mayAutoSign reports whether the transition after previousTransitionHash may be signed with req.workloadOwnerKey.
func mayAutoSign(req *httpSetRequest, previousTransitionHash []byte) (bool, error) {
	if !bytes.Equal(previousTransitionHash, make([]byte, history.HashSize)) {
		return true, nil
	}
	var m manifest.Manifest
	if err := json.Unmarshal(req.manifest, &m); err != nil {
		return false, fmt.Errorf("unmarshaling manifest: %w", err)
	}
	return slices.Contains(m.WorkloadOwnerPubKeys, manifest.MarshalWorkloadOwnerPubKey(&req.workloadOwnerKey.PublicKey)), nil
}

// isRetriableSetError reports whether repeating an identical set request may succeed where err failed.
func isRetriableSetError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// The Coordinator processed the request, but its response was lost while it was read.
	if errors.Is(err, sdk.ErrResponseInterrupted) {
		return true
	}
	if apiErr, ok := errors.AsType[*apitypes.APIError](err); ok {
		return apiErr.StatusCode >= http.StatusInternalServerError
	}
	_, ok := errors.AsType[*url.Error](err)
	return ok
}

// getLatestTransitionV1 returns the hash of the Coordinator's latest transition, or all zeroes if it has no manifest yet.
func getLatestTransitionV1(ctx context.Context, api *apiv1.API) ([]byte, error) {
	nonce, err := cryptohelpers.GenerateRandomBytes(apitypesv1.NonceSize)
	if err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	attestation, err := api.GetAttestation(ctx, nonce)
	if err != nil {
		return nil, fmt.Errorf("getting attestation: %w", err)
	}
	state, err := api.ValidateAttestation(ctx, nonce, attestation)
	switch {
	case errors.Is(err, apiv1.ErrCoordinatorUninitialized):
		return make([]byte, history.HashSize), nil
	case err != nil:
		return nil, fmt.Errorf("validating attestation: %w", err)
	}
	return state.LatestTransitionHash, nil
}

// isDialError reports whether err occurred while connecting, in which case no request was sent.
func isDialError(err error) bool {
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "dial"
}
