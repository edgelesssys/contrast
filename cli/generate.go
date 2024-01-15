package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/edgelesssys/nunki/internal/manifest"
	"github.com/spf13/cobra"
)

const kataPolicyAnnotationKey = "io.katacontainers.config.agent.policy"

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate [flags] paths...",
		Short: "generate",
		RunE:  runGenerate,
	}

	cmd.Flags().StringP("policy", "p", policyDir, "path to policy (.rego) file")
	cmd.Flags().StringP("settings", "s", settingsFilename, "path to settings (.json) file")
	cmd.Flags().StringP("manifest", "m", manifestFilename, "path to manifest (.json) file")

	return cmd
}

func runGenerate(cmd *cobra.Command, args []string) error {
	flags, err := parseGenerateFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	logger, err := newCLILogger(cmd)
	if err != nil {
		return err
	}

	paths, err := findGenerateTargets(args, logger)
	if err != nil {
		return err
	}

	if err := generatePolicies(cmd.Context(), flags.policyPath, flags.settingsPath, paths, logger); err != nil {
		return fmt.Errorf("failed to generate policies: %w", err)
	}

	policies, err := policiesFromKubeResources(paths)
	if err != nil {
		return fmt.Errorf("failed to find kube resources with policy: %w", err)
	}
	policyMap, err := manifestPolicyMapFromPolicies(policies)
	if err != nil {
		return fmt.Errorf("failed to create policy map: %w", err)
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
	manifestData, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(flags.manifestPath, manifestData, 0o644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Updated manifest %s\n", flags.manifestPath)

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

	paths = filterNonCoCoRuntime("kata-cc-isolation", paths, logger)

	if len(paths) == 0 {
		return nil, fmt.Errorf("no .yml/.yaml files found")
	}
	return paths, nil
}

func filterNonCoCoRuntime(runtimeClassName string, paths []string, logger *slog.Logger) []string {
	var filtered []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("Failed to read file", "path", path, "err", err)
			continue
		}
		if !bytes.Contains(data, []byte(runtimeClassName)) {
			logger.Info("Ignoring non-CoCo runtime", "className", runtimeClassName, "path", path)
			continue
		}
		filtered = append(filtered, path)
	}
	return filtered
}

func generatePolicies(ctx context.Context, regoPath, policyPath string, yamlPaths []string, logger *slog.Logger) error {
	if err := createFileWithDefault(filepath.Join(regoPath, policyPath), defaultGenpolicySettings); err != nil {
		return fmt.Errorf("creating default policy file: %w", err)
	}
	if err := createFileWithDefault(filepath.Join(regoPath, rulesFilename), defaultRules); err != nil {
		return fmt.Errorf("creating default policy.rego file: %w", err)
	}
	for _, yamlPath := range yamlPaths {
		policyHash, err := generatePolicyForFile(ctx, regoPath, policyPath, yamlPath, logger)
		if err != nil {
			return fmt.Errorf("failed to generate policy for %s: %w", yamlPath, err)
		}
		if policyHash == [32]byte{} {
			continue
		}
		fmt.Printf("%x  %s\n", policyHash, yamlPath)
	}
	return nil
}

func generatePolicyForFile(ctx context.Context, regoPath, policyPath, yamlPath string, logger *slog.Logger) ([32]byte, error) {
	args := []string{
		"--raw-out",
		"--use-cached-files",
		fmt.Sprintf("--input-files-path=%s", regoPath),
		fmt.Sprintf("--settings-file-name=%s", policyPath),
		fmt.Sprintf("--yaml-file=%s", yamlPath),
	}
	genpolicy := exec.CommandContext(ctx, genpolicyPath, args...)
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

type generateFlags struct {
	policyPath   string
	settingsPath string
	manifestPath string
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
	return &generateFlags{
		policyPath:   policyPath,
		settingsPath: settingsPath,
		manifestPath: manifestPath,
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
func createFileWithDefault(path string, deflt []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(deflt)
	return err
}
