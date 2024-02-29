package cmd

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"slices"
	"time"

	"github.com/edgelesssys/nunki/internal/atls"
	"github.com/edgelesssys/nunki/internal/attestation/snp"
	"github.com/edgelesssys/nunki/internal/fsstore"
	"github.com/edgelesssys/nunki/internal/grpc/dialer"
	"github.com/edgelesssys/nunki/internal/manifest"
	"github.com/edgelesssys/nunki/internal/spinner"
	"github.com/edgelesssys/nunki/internal/userapi"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewSetCmd creates the nunki set subcommand.
func NewSetCmd() *cobra.Command {
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
	cmd.Flags().String("coordinator-policy-hash", DefaultCoordinatorPolicyHash, "expected policy hash of the coordinator, will not be checked if empty")
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

	workloadOwnerKey, err := loadWorkloadOwnerKey(flags.workloadOwnerKeyPath, m, log)
	if err != nil {
		return fmt.Errorf("loading workload owner key: %w", err)
	}

	paths, err := findGenerateTargets(args, log)
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
	log.Debug("Using KDS cache dir", "dir", kdsDir)

	validateOptsGen := newCoordinatorValidateOptsGen(flags.policy)
	kdsCache := fsstore.New(kdsDir, log.WithGroup("kds-cache"))
	kdsGetter := snp.NewCachedHTTPSGetter(kdsCache, snp.NeverGCTicker, log.WithGroup("kds-getter"))
	validator := snp.NewValidator(validateOptsGen, kdsGetter, log.WithGroup("snp-validator"))
	dialer := dialer.NewWithKey(atls.NoIssuer, validator, &net.Dialer{}, workloadOwnerKey)

	conn, err := dialer.Dial(cmd.Context(), flags.coordinator)
	if err != nil {
		return fmt.Errorf("failed to dial coordinator: %w", err)
	}
	defer conn.Close()

	client := userapi.NewUserAPIClient(conn)
	req := &userapi.SetManifestRequest{
		Manifest: manifestBytes,
		Policies: policyMapToBytesList(policies),
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
	manifestPath         string
	coordinator          string
	policy               []byte
	workloadOwnerKeyPath string
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
	policyString, err := cmd.Flags().GetString("coordinator-policy-hash")
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinator-policy-hash flag: %w", err)
	}
	flags.policy, err = hex.DecodeString(policyString)
	if err != nil {
		return nil, fmt.Errorf("hex-decoding coordinator-policy-hash flag: %w", err)
	}
	flags.workloadOwnerKeyPath, err = cmd.Flags().GetString("workload-owner-key")
	if err != nil {
		return nil, fmt.Errorf("getting workload-owner-key flag: %w", err)
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

func loadWorkloadOwnerKey(path string, manifst manifest.Manifest, log *slog.Logger) (*ecdsa.PrivateKey, error) {
	key, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading workload owner key: %w", err)
	}
	pemBlock, _ := pem.Decode(key)
	if pemBlock == nil {
		return nil, fmt.Errorf("decoding workload owner key: %w", err)
	}
	if pemBlock.Type != "EC PRIVATE KEY" {
		return nil, fmt.Errorf("workload owner key is not an EC private key")
	}
	workloadOwnerKey, err := x509.ParseECPrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing workload owner key: %w", err)
	}
	pubKey, err := x509.MarshalPKIXPublicKey(&workloadOwnerKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling public key: %w", err)
	}
	ownerKeyHash := sha256.Sum256(pubKey)
	ownerKeyHex := manifest.NewHexString(ownerKeyHash[:])
	if len(manifst.WorkloadOwnerKeyDigests) == 0 {
		log.Warn("No workload owner keys in manifest. Further manifest updates will be rejected by the coordinator")
		return workloadOwnerKey, nil
	}
	log.Debug("Workload owner keys in manifest", "keys", manifst.WorkloadOwnerKeyDigests)
	if !slices.Contains(manifst.WorkloadOwnerKeyDigests, ownerKeyHex) {
		log.Warn("Workload owner key not found in manifest. This may lock you out from further updates")
	}
	return workloadOwnerKey, nil
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

	var rpcErr error
	for attempts := 0; attempts < 30; attempts++ {
		resp, rpcErr = client.SetManifest(ctx, req)
		if rpcErr == nil {
			return resp, nil
		}
		grpcSt, ok := status.FromError(rpcErr)
		if ok {
			switch grpcSt.Code() {
			case codes.PermissionDenied, codes.InvalidArgument:
				// These errors are not retryable
				return nil, rpcErr
			}
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
