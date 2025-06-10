// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/cli/genpolicy"
	"github.com/edgelesssys/contrast/internal/kubeapi"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"

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
	cmd.SetOut(commandOut())

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
	cmd.Flags().Bool("skip-service-mesh", false, "skip injection of Contrast service mesh sidecar")
	cmd.Flags().StringP("output", "o", "", "output file for generated YAML")
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
		return fmt.Errorf("parse flags: %w", err)
	}

	log, err := newCLILogger(cmd)
	if err != nil {
		return err
	}

	paths, cmPaths, err := findGenerateTargets(args, log)
	if err != nil {
		return err
	}
	if flags.outputFile != "" {
		tmpDir, newPaths, err := getTmpPaths(paths)
		if err != nil {
			return fmt.Errorf("get temporary paths: %w", err)
		}
		defer os.RemoveAll(tmpDir)
		paths = newPaths
	}

	// generate a manifest by checking if a manifest exists and using that,
	// or otherwise using a default.
	var mnf *manifest.Manifest
	existingManifest, err := os.ReadFile(flags.manifestPath)
	if errors.Is(err, fs.ErrNotExist) {
		// Manifest does not exist, create a new one
		mnf, err = manifest.Default(flags.referenceValuesPlatform)
		if err != nil {
			return fmt.Errorf("create default manifest: %w", err)
		}
	} else if err != nil {
		// Manifest exists but could not be read, return error
		return fmt.Errorf("read existing manifest: %w", err)
	} else {
		// Manifest exists and was read successfully, unmarshal and validate it
		if err := json.Unmarshal(existingManifest, &mnf); err != nil {
			return fmt.Errorf("unmarshal existing manifest: %w", err)
		}
		if err := mnf.Validate(); err != nil {
			return fmt.Errorf("validate existing manifest: %w", err)
		}
	}

	runtimeHandler, err := manifest.RuntimeHandler(flags.referenceValuesPlatform)
	if err != nil {
		return fmt.Errorf("get runtime handler: %w", err)
	}

	if err := patchTargets(paths, flags.imageReplacementsFile, runtimeHandler, flags.skipInitializer, flags.skipServiceMesh, log); err != nil {
		return fmt.Errorf("patch targets: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Patched targets")

	if err := generatePolicies(cmd.Context(), flags, paths, cmPaths, log); err != nil {
		return fmt.Errorf("generate policies: %w", err)
	}

	if flags.outputFile != "" {
		combinedYAML, err := getCombinedYAML(paths)
		if err != nil {
			return fmt.Errorf("get combined YAML: %w", err)
		}
		if err := os.WriteFile(flags.outputFile, combinedYAML, 0o644); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Generated workload policy annotations")

	policies, err := policiesFromKubeResources(paths)
	if err != nil {
		return fmt.Errorf("find kube resources with policy: %w", err)
	}
	policyMap, err := manifestPolicyMapFromPolicies(policies)
	if err != nil {
		return fmt.Errorf("create policy map: %w", err)
	}

	if err := generateWorkloadOwnerKey(flags); err != nil {
		return fmt.Errorf("generating workload owner key: %w", err)
	}
	if err := generateSeedshareOwnerKey(flags); err != nil {
		return fmt.Errorf("generating seedshare owner key: %w", err)
	}

	mnf.Policies = policyMap
	// Existing manifests are already validated above, but newly generated manifests may be missing reference values or a coordinator.
	var ce *manifest.CoordinatorCountError
	var ve *manifest.ValidationError
	if err := mnf.Validate(); errors.As(err, &ce) {
		if ce.Count == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "  No Coordinator resource found, did you forget to add it to your resources?")
		}
		return ce
	} else if errors.As(err, &ve) && ve.OnlyExpectedMissingReferenceValues() {
		fmt.Fprintf(cmd.OutOrStdout(), "  Please fill in the reference values for %s\n", flags.referenceValuesPlatform.String())
	} else if err != nil {
		return err
	}

	if flags.disableUpdates {
		mnf.WorkloadOwnerKeyDigests = nil
	} else {
		for _, keyPath := range flags.workloadOwnerKeys {
			if err := addWorkloadOwnerKeyToManifest(mnf, keyPath); err != nil {
				return fmt.Errorf("adding workload owner key to manifest: %w", err)
			}
		}
	}
	slices.Sort(mnf.WorkloadOwnerKeyDigests)

	for _, keyPath := range flags.seedshareOwnerKeys {
		if err := addSeedshareOwnerKeyToManifest(mnf, keyPath); err != nil {
			return fmt.Errorf("adding seedshare owner key to manifest: %w", err)
		}
	}
	slices.Sort(mnf.SeedshareOwnerPubKeys)

	manifestData, err := json.MarshalIndent(mnf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(flags.manifestPath, append(manifestData, '\n'), 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✔️ Updated manifest %s\n", flags.manifestPath)
	return nil
}

func findGenerateTargets(args []string, logger *slog.Logger) ([]string, []string, error) {
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
			return nil, nil, fmt.Errorf("walk %s: %w", path, err)
		}
	}
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("no .yml/.yaml files found")
	}

	cmPaths := filterForConfigMaps(paths, logger)

	paths = filterNonCoCoRuntime("contrast-cc", paths, logger)
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("no .yml/.yaml files with 'contrast-cc' runtime found")
	}

	return paths, cmPaths, nil
}

func filterForConfigMaps(paths []string, logger *slog.Logger) []string {
	var filtered []string

pathLoop:
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("Could not read file", "path", path, "err", err)
			continue
		}
		objects, err := kubeapi.UnmarshalUnstructuredK8SResource(data)
		if err != nil {
			logger.Warn("Could not parse file into Kubernetes resources", "path", path, "err", err)
			continue
		}
		for _, object := range objects {
			if object.GetKind() == "ConfigMap" || object.GetKind() == "Secret" {
				filtered = append(filtered, path)
				continue pathLoop
			}
		}
		logger.Debug("File without ConfigMap and Secret", "path", path)
	}
	return filtered
}

func filterNonCoCoRuntime(runtimeClassNamePrefix string, paths []string, logger *slog.Logger) []string {
	var filtered []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("read file", "path", path, "err", err)
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

func generatePolicies(ctx context.Context, flags *generateFlags, yamlPaths, cmPaths []string, logger *slog.Logger) error {
	cfg := genpolicy.NewConfig(flags.referenceValuesPlatform)
	if err := createFileWithDefault(flags.settingsPath, 0o644, func() ([]byte, error) { return cfg.Settings, nil }); err != nil {
		return fmt.Errorf("creating default policy file: %w", err)
	}
	if err := createFileWithDefault(flags.policyPath, 0o644, func() ([]byte, error) { return cfg.Rules, nil }); err != nil {
		return fmt.Errorf("creating default policy.rego file: %w", err)
	}

	runner, err := genpolicy.New(flags.policyPath, flags.settingsPath, flags.genpolicyCachePath, cfg.Bin)
	if err != nil {
		return fmt.Errorf("preparing genpolicy: %w", err)
	}

	defer func() {
		if err := runner.Teardown(); err != nil {
			logger.Warn("Cleanup failed", "err", err)
		}
	}()

	for _, yamlPath := range yamlPaths {
		if err := runner.Run(ctx, yamlPath, cmPaths, logger); err != nil {
			return fmt.Errorf("failed to generate policy for %s: %w", yamlPath, err)
		}
	}
	return nil
}

func patchTargets(paths []string, imageReplacementsFile, runtimeHandler string, skipInitializer, skipServiceMesh bool, logger *slog.Logger) error {
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
			return fmt.Errorf("read %s: %w", path, err)
		}
		kubeObjs, err := kuberesource.UnmarshalApplyConfigurations(data)
		if err != nil {
			return fmt.Errorf("unmarshal %s: %w", path, err)
		}

		if !skipInitializer {
			if err := injectInitializer(kubeObjs); err != nil {
				return fmt.Errorf("injecting Initializer: %w", err)
			}
		}
		if !skipServiceMesh {
			if err := injectServiceMesh(kubeObjs); err != nil {
				return fmt.Errorf("injecting Service Mesh: %w", err)
			}
		}

		kubeObjs = kuberesource.PatchImages(kubeObjs, replacements)

		replaceRuntimeClassName := runtimeClassNamePatcher(runtimeHandler)
		for i := range kubeObjs {
			kubeObjs[i] = kuberesource.MapPodSpec(kubeObjs[i], replaceRuntimeClassName)
		}

		logger.Debug("Updating resources in yaml file", "path", path)
		resource, err := kuberesource.EncodeResources(kubeObjs...)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, resource, 0o666); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	return nil
}

func injectInitializer(resources []any) error {
	for _, resource := range resources {
		switch r := resource.(type) {
		case *applyappsv1.StatefulSetApplyConfiguration:
			if r.Spec != nil && r.Spec.Template != nil && r.Spec.Template.ObjectMetaApplyConfiguration != nil && r.Spec.Template.Annotations != nil &&
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
		r, ok := resource.(*applyappsv1.StatefulSetApplyConfiguration)
		if ok && r.Spec != nil && r.Spec.Template != nil && r.Spec.Template.ObjectMetaApplyConfiguration != nil && r.Spec.Template.Annotations != nil &&
			r.Spec.Template.Annotations[contrastRoleAnnotationKey] == "coordinator" {
			continue
		}
		_, err := kuberesource.AddServiceMesh(resource, kuberesource.ServiceMeshProxy())
		if err != nil {
			return err
		}
	}
	return nil
}

func runtimeClassNamePatcher(handler string) func(*applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
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

func validateOutputFile(outputFile string) error {
	if outputFile == "" {
		return nil
	}
	dir := filepath.Dir(outputFile)
	if stat, err := os.Stat(dir); err != nil {
		return err
	} else if !stat.IsDir() {
		return fmt.Errorf("not a directory: %s", dir)
	}
	if fi, err := os.Stat(outputFile); err == nil && fi.IsDir() {
		return fmt.Errorf("output file %s is a directory", outputFile)
	}
	return nil
}

func getTmpPaths(paths []string) (string, []string, error) {
	tmpDir, err := os.MkdirTemp("", "contrast-generate")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary directory: %w", err)
	}
	var newPaths []string
	for _, path := range paths {
		if err := copyFile(path, filepath.Join(tmpDir, filepath.Base(path))); err != nil {
			return tmpDir, nil, fmt.Errorf("copy %s: %w", path, err)
		}
		newPaths = append(newPaths, filepath.Join(tmpDir, filepath.Base(path)))
	}
	return tmpDir, newPaths, nil
}

func copyFile(inPath, outPath string) error {
	inFile, err := os.Open(inPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", inPath, err)
	}
	defer inFile.Close()

	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outPath, err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, inFile); err != nil {
		return fmt.Errorf("copy %s: %w", inPath, err)
	}
	return nil
}

func getCombinedYAML(paths []string) ([]byte, error) {
	var combinedYAML []byte
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		// This expects a "---" separator at the beginning of each YAML file,
		// as is the case after running "genpolicy".
		combinedYAML = append(combinedYAML, data...)
	}
	return combinedYAML, nil
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
	skipServiceMesh         bool
	outputFile              string
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
	skipServiceMesh, err := cmd.Flags().GetBool("skip-service-mesh")
	if err != nil {
		return nil, err
	}
	outputFile, err := cmd.Flags().GetString("output")
	if err != nil {
		return nil, err
	}
	if err := validateOutputFile(outputFile); err != nil {
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
		skipServiceMesh:         skipServiceMesh,
		outputFile:              outputFile,
	}, nil
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
