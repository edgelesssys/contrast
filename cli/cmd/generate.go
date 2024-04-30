// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/internal/embedbin"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/spf13/cobra"
)

const (
	kataPolicyAnnotationKey   = "io.katacontainers.config.agent.policy"
	contrastRoleAnnotationKey = "contrast.edgeless.systems/pod-role"
)

// NewGenerateCmd creates the contrast generate subcommand.
func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate [flags] paths...",
		Short: "generate policies and inject into Kubernetes resources",
		Long: `Generate policies and inject into the given Kubernetes resources.

This will download the referenced container images to calculate the dm-verity
hashes of the image layers. In addition, the Rego policy will be used as base
and updated with the given settings file. For each container workload, the policy
is added as an annotation to the Kubernetes YAML.

The hashes of the policies are added to the manifest.

If the Kubernetes YAML contains a Contrast Coordinator pod whose policy differs from
the embedded default, the generated policy will be printed to stdout, alongside a
warning message on stderr. This hash needs to be passed to the set and verify
subcommands.`,
		RunE: withTelemetry(runGenerate),
	}

	cmd.Flags().StringP("policy", "p", rulesFilename, "path to policy (.rego) file")
	cmd.Flags().StringP("settings", "s", settingsFilename, "path to settings (.json) file")
	cmd.Flags().StringP("manifest", "m", manifestFilename, "path to manifest (.json) file")
	cmd.Flags().StringArrayP("workload-owner-key", "w", []string{workloadOwnerPEM}, "path to workload owner key (.pem) file")
	cmd.Flags().BoolP("disable-updates", "d", false, "prevent further updates of the manifest")
	must(cmd.MarkFlagFilename("policy", "rego"))
	must(cmd.MarkFlagFilename("settings", "json"))
	must(cmd.MarkFlagFilename("manifest", "json"))
	cmd.MarkFlagsMutuallyExclusive("workload-owner-key", "disable-updates")
	return cmd
}

func runGenerate(cmd *cobra.Command, args []string) error {
	flags, err := parseGenerateFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	log, err := newCLILogger(cmd)
	if err != nil {
		return err
	}

	paths, err := findGenerateTargets(args, log)
	if err != nil {
		return err
	}

	if err := generatePolicies(cmd.Context(), flags.policyPath, flags.settingsPath, paths, log); err != nil {
		return fmt.Errorf("failed to generate policies: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Generated workload policy annotations")

	policies, err := policiesFromKubeResources(paths)
	if err != nil {
		return fmt.Errorf("failed to find kube resources with policy: %w", err)
	}
	policyMap, err := manifestPolicyMapFromPolicies(policies)
	if err != nil {
		return fmt.Errorf("failed to create policy map: %w", err)
	}

	if err := generateWorkloadOwnerKey(flags); err != nil {
		return fmt.Errorf("generating workload owner key: %w", err)
	}

	defaultManifest := manifest.Default()
	defaultManifestData, err := json.MarshalIndent(&defaultManifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling default manifest: %w", err)
	}
	manifestData, err := readFileOrDefault(flags.manifestPath, defaultManifestData)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}
	var manifest *manifest.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}
	manifest.Policies = policyMap

	if flags.disableUpdates {
		manifest.WorkloadOwnerKeyDigests = nil
	} else {
		for _, keyPath := range flags.workloadOwnerKeys {
			if err := addWorkloadOwnerKeyToManifest(manifest, keyPath); err != nil {
				return fmt.Errorf("adding workload owner key to manifest: %w", err)
			}
		}
	}
	slices.Sort(manifest.WorkloadOwnerKeyDigests)

	manifestData, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(flags.manifestPath, manifestData, 0o644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✔️ Updated manifest %s\n", flags.manifestPath)

	if hash := getCoordinatorPolicyHash(policies, log); hash != "" {
		coordHashPath := filepath.Join(flags.workspaceDir, coordHashFilename)
		if err := os.WriteFile(coordHashPath, []byte(hash), 0o644); err != nil {
			return fmt.Errorf("failed to write coordinator policy hash: %w", err)
		}
	}

	return nil
}

func findGenerateTargets(args []string, logger *slog.Logger) ([]string, error) {
	var paths []string
	for _, path := range args {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil // Skip directories
			}
			switch {
			case strings.HasSuffix(info.Name(), ".yaml"):
				paths = append(paths, path)
			case strings.HasSuffix(info.Name(), ".yml"):
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk %s: %w", path, err)
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no .yml/.yaml files found")
	}

	paths = filterNonCoCoRuntime("contrast-cc", paths, logger)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no .yml/.yaml files with 'contrast-cc' runtime found")
	}

	return paths, nil
}

func filterNonCoCoRuntime(runtimeClassNamePrefix string, paths []string, logger *slog.Logger) []string {
	var filtered []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("Failed to read file", "path", path, "err", err)
			continue
		}
		if !bytes.Contains(data, []byte(runtimeClassNamePrefix)) {
			logger.Info("Ignoring non-CoCo runtime", "className", runtimeClassNamePrefix, "path", path)
			continue
		}
		filtered = append(filtered, path)
	}
	return filtered
}

func generatePolicies(ctx context.Context, regoRulesPath, policySettingsPath string, yamlPaths []string, logger *slog.Logger) error {
	if err := createFileWithDefault(policySettingsPath, func() ([]byte, error) { return defaultGenpolicySettings, nil }); err != nil {
		return fmt.Errorf("creating default policy file: %w", err)
	}
	if err := createFileWithDefault(regoRulesPath, func() ([]byte, error) { return defaultRules, nil }); err != nil {
		return fmt.Errorf("creating default policy.rego file: %w", err)
	}
	binaryInstallDir, err := installDir()
	if err != nil {
		return fmt.Errorf("failed to get install dir: %w", err)
	}
	genpolicyInstall, err := embedbin.New().Install(binaryInstallDir, genpolicyBin)
	if err != nil {
		return fmt.Errorf("failed to install genpolicy: %w", err)
	}
	defer func() {
		if err := genpolicyInstall.Uninstall(); err != nil {
			logger.Warn("Failed to uninstall genpolicy tool", "err", err)
		}
	}()
	for _, yamlPath := range yamlPaths {
		policyHash, err := generatePolicyForFile(ctx, genpolicyInstall.Path(), regoRulesPath, policySettingsPath, yamlPath, logger)
		if err != nil {
			return fmt.Errorf("failed to generate policy for %s: %w", yamlPath, err)
		}
		if policyHash == [32]byte{} {
			continue
		}

		logger.Info("Calculated policy hash", "hash", hex.EncodeToString(policyHash[:]), "path", yamlPath)
	}
	return nil
}

func addWorkloadOwnerKeyToManifest(manifst *manifest.Manifest, keyPath string) error {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("reading workload owner key: %w", err)
	}
	block, _ := pem.Decode(keyData)
	if block == nil {
		return errors.New("failed to decode PEM block")
	}
	var publicKey []byte
	switch block.Type {
	case "PUBLIC KEY":
		publicKey = block.Bytes
	case "EC PRIVATE KEY":
		privateKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("parsing EC private key: %w", err)
		}
		publicKey, err = x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		if err != nil {
			return fmt.Errorf("marshaling public key: %w", err)
		}
	default:
		return fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}

	hash := sha256.Sum256(publicKey)
	hashString := manifest.NewHexString(hash[:])
	for _, existingHash := range manifst.WorkloadOwnerKeyDigests {
		if existingHash == hashString {
			return nil
		}
	}
	manifst.WorkloadOwnerKeyDigests = append(manifst.WorkloadOwnerKeyDigests, hashString)
	return nil
}

func generatePolicyForFile(ctx context.Context, genpolicyPath, regoPath, policyPath, yamlPath string, logger *slog.Logger) ([32]byte, error) {
	args := []string{
		"--raw-out",
		"--use-cached-files",
		fmt.Sprintf("--rego-rules-path=%s", regoPath),
		fmt.Sprintf("--json-settings-path=%s", policyPath),
		fmt.Sprintf("--yaml-file=%s", yamlPath),
	}
	genpolicy := exec.CommandContext(ctx, genpolicyPath, args...)
	genpolicy.Env = append(genpolicy.Env, "RUST_LOG=DEBUG")
	var stdout, stderr bytes.Buffer
	genpolicy.Stdout = &stdout
	genpolicy.Stderr = &stderr
	if err := genpolicy.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return [32]byte{}, fmt.Errorf("genpolicy failed with exit code %d: %s",
				exitErr.ExitCode(), stderr.String())
		}
		return [32]byte{}, fmt.Errorf("genpolicy failed: %w", err)
	}
	if stdout.Len() == 0 {
		logger.Info("Policy output is empty, ignoring the file", "yamlPath", yamlPath)
		return [32]byte{}, nil
	}
	policyHash := sha256.Sum256(stdout.Bytes())
	return policyHash, nil
}

func generateWorkloadOwnerKey(flags *generateFlags) error {
	if flags.disableUpdates || len(flags.workloadOwnerKeys) != 1 {
		// No need to generate keys
		// either updates are disabled or
		// the user has provided a set of (presumably already generated) public keys
		return nil
	}
	keyPath := flags.workloadOwnerKeys[0]

	if err := createFileWithDefault(keyPath, newKeyPair); err != nil {
		return fmt.Errorf("creating default workload owner key file: %w", err)
	}
	return nil
}

func newKeyPair() ([]byte, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating private key: %w", err)
	}
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}), nil
}

type generateFlags struct {
	policyPath        string
	settingsPath      string
	manifestPath      string
	workloadOwnerKeys []string
	disableUpdates    bool
	workspaceDir      string
}

func parseGenerateFlags(cmd *cobra.Command) (*generateFlags, error) {
	policyPath, err := cmd.Flags().GetString("policy")
	if err != nil {
		return nil, err
	}
	settingsPath, err := cmd.Flags().GetString("settings")
	if err != nil {
		return nil, err
	}
	manifestPath, err := cmd.Flags().GetString("manifest")
	if err != nil {
		return nil, err
	}
	workloadOwnerKeys, err := cmd.Flags().GetStringArray("workload-owner-key")
	if err != nil {
		return nil, err
	}
	disableUpdates, err := cmd.Flags().GetBool("disable-updates")
	if err != nil {
		return nil, err
	}
	workspaceDir, err := cmd.Flags().GetString("workspace-dir")
	if err != nil {
		return nil, err
	}
	if workspaceDir != "" {
		// Prepend default paths with workspaceDir
		if !cmd.Flags().Changed("settings") {
			settingsPath = filepath.Join(workspaceDir, settingsFilename)
		}
		if !cmd.Flags().Changed("policy") {
			policyPath = filepath.Join(workspaceDir, rulesFilename)
		}
		if !cmd.Flags().Changed("manifest") {
			manifestPath = filepath.Join(workspaceDir, manifestFilename)
		}
		if !cmd.Flags().Changed("workload-owner-key") {
			workloadOwnerKeys = []string{filepath.Join(workspaceDir, workloadOwnerKeys[0])}
		}
	}

	return &generateFlags{
		policyPath:        policyPath,
		settingsPath:      settingsPath,
		manifestPath:      manifestPath,
		workloadOwnerKeys: workloadOwnerKeys,
		disableUpdates:    disableUpdates,
		workspaceDir:      workspaceDir,
	}, nil
}

// readFileOrDefault reads the file at path,
// or returns the default value if the file doesn't exist.
func readFileOrDefault(path string, deflt []byte) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	return deflt, nil
}

// createFileWithDefault creates the file at path with the default value,
// if it doesn't exist.
func createFileWithDefault(path string, dflt func() ([]byte, error)) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	content, err := dflt()
	if err != nil {
		return err
	}
	_, err = file.Write(content)
	return err
}

func installDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".contrast"), nil
}
