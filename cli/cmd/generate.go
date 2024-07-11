// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/internal/embedbin"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/node-installer/platforms"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"

	"github.com/spf13/cobra"
)

const (
	kataPolicyAnnotationKey      = "io.katacontainers.config.agent.policy"
	contrastRoleAnnotationKey    = "contrast.edgeless.systems/pod-role"
	skipInitializerAnnotationKey = "contrast.edgeless.systems/skip-initializer"
)

// NewGenerateCmd creates the contrast generate subcommand.
func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate [flags] paths...",
		Short: "generate policies and inject into Kubernetes resources",
		Long: `Generate policies and inject into the given Kubernetes resources.

This will add the Contrast Initializer and Contrast Service Mesh as init containers
to your workloads and then download the referenced container images to calculate the
dm-verity hashes of the image layers. In addition, the Rego policy will be used as
base and updated with the given settings file. For each container workload, the
policy is added as an annotation to the Kubernetes YAML.

The hashes of the policies are added to the manifest.

If the Kubernetes YAML contains a Contrast Coordinator pod whose policy differs from
the embedded default, the generated policy will be printed to stdout, alongside a
warning message on stderr. This hash needs to be passed to the set and verify
subcommands.`,
		RunE: withTelemetry(runGenerate),
	}

	cmd.Flags().StringP("policy", "p", rulesFilename, "path to policy (.rego) file")
	cmd.Flags().StringP("settings", "s", settingsFilename, "path to settings (.json) file")
	cmd.Flags().StringP("genpolicy-cache-path", "c", layersCacheFilename, "path to cache for the cache (.json) file containing the image layers")
	cmd.Flags().StringP("manifest", "m", manifestFilename, "path to manifest (.json) file")
	cmd.Flags().String("reference-values", "",
		fmt.Sprintf("set the default reference values used for attestation (one of: %s)",
			strings.Join(platforms.AllStrings(), ", "),
		),
	)
	must(cmd.MarkFlagRequired("reference-values"))
	cmd.Flags().StringArrayP("add-workload-owner-key", "w", []string{workloadOwnerPEM},
		"add a workload owner key from a PEM file to the manifest (pass more than once to add multiple keys)")
	cmd.Flags().StringArray("add-seedshare-owner-key", []string{seedshareOwnerPEM},
		"add a seedshare owner key from a PEM file to the manifest (pass more than once to add multiple keys)")
	cmd.Flags().BoolP("disable-updates", "d", false, "prevent further updates of the manifest")
	cmd.Flags().String("image-replacements", "", "path to image replacements file")
	cmd.Flags().Bool("skip-initializer", false, "skip injection of Contrast Initializer")
	must(cmd.Flags().MarkHidden("image-replacements"))
	must(cmd.MarkFlagFilename("policy", "rego"))
	must(cmd.MarkFlagFilename("settings", "json"))
	must(cmd.MarkFlagFilename("manifest", "json"))
	cmd.MarkFlagsMutuallyExclusive("add-workload-owner-key", "disable-updates")
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

	if err := patchTargets(paths, flags.imageReplacementsFile, flags.skipInitializer, log); err != nil {
		return fmt.Errorf("failed to patch targets: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Patched targets")

	if err := generatePolicies(cmd.Context(), flags.policyPath, flags.settingsPath, flags.genpolicyCachePath, paths, log); err != nil {
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
	if err := generateSeedshareOwnerKey(flags); err != nil {
		return fmt.Errorf("generating seedshare owner key: %w", err)
	}

	defaultManifest := manifest.Default()
	switch flags.referenceValuesPlatform {
	case platforms.AKSCloudHypervisorSNP:
		defaultManifest = manifest.DefaultAKS()
	}

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
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("validating manifest: %w", err)
	}

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

	for _, keyPath := range flags.seedshareOwnerKeys {
		if err := addSeedshareOwnerKeyToManifest(manifest, keyPath); err != nil {
			return fmt.Errorf("adding seedshare owner key to manifest: %w", err)
		}
	}
	slices.Sort(manifest.SeedshareOwnerPubKeys)

	manifestData, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(flags.manifestPath, append(manifestData, '\n'), 0o644); err != nil {
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

func generatePolicies(ctx context.Context, regoRulesPath, policySettingsPath, genpolicyCachePath string, yamlPaths []string, logger *slog.Logger) error {
	if err := createFileWithDefault(policySettingsPath, 0o644, func() ([]byte, error) { return defaultGenpolicySettings, nil }); err != nil {
		return fmt.Errorf("creating default policy file: %w", err)
	}
	if err := createFileWithDefault(regoRulesPath, 0o644, func() ([]byte, error) { return defaultRules, nil }); err != nil {
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
		policyHash, err := generatePolicyForFile(ctx, genpolicyInstall.Path(), regoRulesPath, policySettingsPath, yamlPath, genpolicyCachePath, logger)
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

func patchTargets(paths []string, imageReplacementsFile string, skipInitializer bool, logger *slog.Logger) error {
	var replacements map[string]string
	var err error
	if imageReplacementsFile != "" {
		f, err := os.Open(imageReplacementsFile)
		if err != nil {
			return fmt.Errorf("opening image replacements file %s: %w", imageReplacementsFile, err)
		}
		defer f.Close()

		replacements, err = kuberesource.ImageReplacementsFromFile(f)
		if err != nil {
			return fmt.Errorf("parsing image definition file %s: %w", imageReplacementsFile, err)
		}
	} else {
		replacements, err = kuberesource.ImageReplacementsFromFile(bytes.NewReader(ReleaseImageReplacements))
		if err != nil {
			return fmt.Errorf("parsing release image definitions %s: %w", ReleaseImageReplacements, err)
		}
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		kubeObjs, err := kuberesource.UnmarshalApplyConfigurations(data)
		if err != nil {
			return fmt.Errorf("failed to unmarshal %s: %w", path, err)
		}

		if !skipInitializer {
			if err := injectInitializer(kubeObjs); err != nil {
				return fmt.Errorf("injecting Initializer: %w", err)
			}
		}
		if err := injectServiceMesh(kubeObjs); err != nil {
			return fmt.Errorf("injecting Service Mesh: %w", err)
		}

		kubeObjs = kuberesource.PatchImages(kubeObjs, replacements)

		replaceRuntimeClassName := runtimeClassNamePatcher()
		for i := range kubeObjs {
			kubeObjs[i] = kuberesource.MapPodSpec(kubeObjs[i], replaceRuntimeClassName)
		}

		logger.Debug("Updating resources in yaml file", "path", path)
		resource, err := kuberesource.EncodeResources(kubeObjs...)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, resource, os.ModePerm); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}
	return nil
}

func injectInitializer(resources []any) error {
	for _, resource := range resources {
		switch r := resource.(type) {
		case *applyappsv1.StatefulSetApplyConfiguration:
			if r.Spec != nil && r.Spec.Template != nil &&
				r.Spec.Template.Annotations[contrastRoleAnnotationKey] == "coordinator" {
				continue
			}
		}
		_, err := kuberesource.AddInitializer(resource, kuberesource.Initializer())
		if err != nil {
			return err
		}
	}
	return nil
}

func injectServiceMesh(resources []any) error {
	for _, resource := range resources {
		deploy, ok := resource.(*applyappsv1.StatefulSetApplyConfiguration)
		if ok && deploy.Spec.Template.Annotations[contrastRoleAnnotationKey] == "coordinator" {
			continue
		}
		_, err := kuberesource.AddServiceMesh(resource, kuberesource.ServiceMeshProxy())
		if err != nil {
			return err
		}
	}
	return nil
}

func runtimeClassNamePatcher() func(*applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
	handler := runtimeHandler(manifest.TrustedMeasurement)
	return func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		if spec.RuntimeClassName == nil || *spec.RuntimeClassName == handler {
			return spec
		}

		if strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") || *spec.RuntimeClassName == "kata-cc-isolation" {
			spec.RuntimeClassName = &handler
		}
		return spec
	}
}

func addWorkloadOwnerKeyToManifest(manifst *manifest.Manifest, keyPath string) error {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("reading workload owner key: %w", err)
	}
	publicKey, err := manifest.ExtractWorkloadOwnerPublicKey(keyData)
	if err != nil {
		return fmt.Errorf("reading workload owner key: %w", err)
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

func addSeedshareOwnerKeyToManifest(manifst *manifest.Manifest, keyPath string) error {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("reading seedshare owner key: %w", err)
	}
	publicKey, err := manifest.ExtractSeedshareOwnerPublicKey(keyData)
	if err != nil {
		return fmt.Errorf("extracting seed share public key: %w", err)
	}
	if !slices.Contains(manifst.SeedshareOwnerPubKeys, publicKey) {
		manifst.SeedshareOwnerPubKeys = append(manifst.SeedshareOwnerPubKeys, publicKey)
	}

	return nil
}

type logTranslator struct {
	r         *io.PipeReader
	w         *io.PipeWriter
	logger    *slog.Logger
	stopDoneC chan struct{}
}

func newLogTranslator(logger *slog.Logger) logTranslator {
	r, w := io.Pipe()
	l := logTranslator{
		r:         r,
		w:         w,
		logger:    logger,
		stopDoneC: make(chan struct{}),
	}
	l.startTranslate()
	return l
}

func (l logTranslator) Write(p []byte) (n int, err error) {
	return l.w.Write(p)
}

var genpolicyLogPrefixReg = regexp.MustCompile(`^\[[^\]\s]+\s+(\w+)\s+([^\]\s]+)\] (.*)`)

func (l logTranslator) startTranslate() {
	go func() {
		defer close(l.stopDoneC)
		scanner := bufio.NewScanner(l.r)
		for scanner.Scan() {
			line := scanner.Text()
			match := genpolicyLogPrefixReg.FindStringSubmatch(line)
			if len(match) != 4 {
				// genpolicy prints some warnings without the logger
				l.logger.Warn(line)
			} else {
				switch match[1] {
				case "ERROR":
					l.logger.Error(match[3], "position", match[2])
				case "WARN":
					l.logger.Warn(match[3], "position", match[2])
				case "INFO": // prints quite a lot, only show on debug
					l.logger.Debug(match[3], "position", match[2])
				}
			}
		}
	}()
}

func (l logTranslator) stop() {
	l.w.Close()
	<-l.stopDoneC
}

func generatePolicyForFile(ctx context.Context, genpolicyPath, regoPath, policyPath, yamlPath, genpolicyCachePath string, logger *slog.Logger) ([32]byte, error) {
	args := []string{
		"--raw-out",
		fmt.Sprintf("--runtime-class-names=%s", "contrast-cc"),
		fmt.Sprintf("--rego-rules-path=%s", regoPath),
		fmt.Sprintf("--json-settings-path=%s", policyPath),
		fmt.Sprintf("--yaml-file=%s", yamlPath),
		fmt.Sprintf("--layers-cache-file-path=%s", genpolicyCachePath),
	}
	genpolicy := exec.CommandContext(ctx, genpolicyPath, args...)
	genpolicy.Env = append(genpolicy.Env, "RUST_LOG=info", "RUST_BACKTRACE=1")

	logFilter := newLogTranslator(logger)
	defer logFilter.stop()
	var stdout bytes.Buffer
	genpolicy.Stdout = &stdout
	genpolicy.Stderr = logFilter

	if err := genpolicy.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return [32]byte{}, fmt.Errorf("genpolicy failed with exit code %d", exitErr.ExitCode())
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

	if err := createFileWithDefault(keyPath, 0o600, manifest.NewWorkloadOwnerKey); err != nil {
		return fmt.Errorf("creating default workload owner key file: %w", err)
	}
	return nil
}

func generateSeedshareOwnerKey(flags *generateFlags) error {
	if len(flags.seedshareOwnerKeys) != 1 {
		// No need to generate keys
		// the user has provided a set of (presumably already generated) public keys
		return nil
	}
	keyPath := flags.seedshareOwnerKeys[0]

	if err := createFileWithDefault(keyPath, 0o600, manifest.NewSeedShareOwnerPrivateKey); err != nil {
		return fmt.Errorf("creating default seedshare owner key file: %w", err)
	}
	return nil
}

type generateFlags struct {
	policyPath              string
	settingsPath            string
	manifestPath            string
	genpolicyCachePath      string
	referenceValuesPlatform platforms.Platform
	workloadOwnerKeys       []string
	seedshareOwnerKeys      []string
	disableUpdates          bool
	workspaceDir            string
	imageReplacementsFile   string
	skipInitializer         bool
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
	genpolicyCachePath, err := cmd.Flags().GetString("genpolicy-cache-path")
	if err != nil {
		return nil, err
	}
	manifestPath, err := cmd.Flags().GetString("manifest")
	if err != nil {
		return nil, err
	}
	referenceValues, err := cmd.Flags().GetString("reference-values")
	if err != nil {
		return nil, err
	}
	referenceValuesPlatform, err := platforms.FromString(referenceValues)
	if err != nil {
		return nil, fmt.Errorf("invalid reference-values platform: %w", err)
	}
	workloadOwnerKeys, err := cmd.Flags().GetStringArray("add-workload-owner-key")
	if err != nil {
		return nil, err
	}
	seedshareOwnerKeys, err := cmd.Flags().GetStringArray("add-seedshare-owner-key")
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
		if !cmd.Flags().Changed("genpolicy-cache-path") {
			genpolicyCachePath = filepath.Join(workspaceDir, genpolicyCachePath)
		}
		if !cmd.Flags().Changed("policy") {
			policyPath = filepath.Join(workspaceDir, rulesFilename)
		}
		if !cmd.Flags().Changed("manifest") {
			manifestPath = filepath.Join(workspaceDir, manifestFilename)
		}
		if !cmd.Flags().Changed("add-workload-owner-key") {
			workloadOwnerKeys = []string{filepath.Join(workspaceDir, workloadOwnerKeys[0])}
		}
		if !cmd.Flags().Changed("add-seedshare-owner-key") {
			seedshareOwnerKeys = []string{filepath.Join(workspaceDir, seedshareOwnerKeys[0])}
		}
	}

	imageReplacementsFile, err := cmd.Flags().GetString("image-replacements")
	if err != nil {
		return nil, err
	}

	skipInitializer, err := cmd.Flags().GetBool("skip-initializer")
	if err != nil {
		return nil, err
	}

	return &generateFlags{
		policyPath:              policyPath,
		settingsPath:            settingsPath,
		genpolicyCachePath:      genpolicyCachePath,
		manifestPath:            manifestPath,
		referenceValuesPlatform: referenceValuesPlatform,
		workloadOwnerKeys:       workloadOwnerKeys,
		seedshareOwnerKeys:      seedshareOwnerKeys,
		disableUpdates:          disableUpdates,
		workspaceDir:            workspaceDir,
		imageReplacementsFile:   imageReplacementsFile,
		skipInitializer:         skipInitializer,
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
func createFileWithDefault(path string, perm fs.FileMode, dflt func() ([]byte, error)) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
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
