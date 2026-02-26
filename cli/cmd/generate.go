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
	"strconv"
	"strings"

	"github.com/edgelesssys/contrast/cli/genpolicy"
	"github.com/edgelesssys/contrast/cli/verifier"
	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"

	"github.com/spf13/cobra"
)

const (
	contrastRoleAnnotationKey     = "contrast.edgeless.systems/pod-role"
	workloadSecretIDAnnotationKey = "contrast.edgeless.systems/workload-secret-id"
	hypervisorCPUCountAnnotation  = "io.katacontainers.config.hypervisor.default_vcpus"
	idBlockAnnotation             = "contrast.edgeless.systems/snp-id-block/"
	amdCPUGenerationMilan         = "Milan"
	amdCPUGenerationGenoa         = "Genoa"
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
	cmd.Flags().StringArray("reference-value-patches", []string{},
		"add reference value patches to apply to the reference values (pass more than once to add multiple patch files)")
	must(cmd.Flags().MarkHidden("reference-value-patches"))
	cmd.Flags().Bool("purge-empty-reference-values", false, "purge reference values with missing values from the manifest. Caution advised!")
	must(cmd.Flags().MarkHidden("purge-empty-reference-values"))
	cmd.Flags().StringArrayP("add-workload-owner-key", "w", []string{workloadOwnerPEM},
		"add a workload owner key from a PEM file to the manifest (pass more than once to add multiple keys)")
	cmd.Flags().StringArray("add-seedshare-owner-key", []string{seedshareOwnerPEM},
		"add a seedshare owner key from a PEM file to the manifest (pass more than once to add multiple keys)")
	cmd.Flags().BoolP("disable-updates", "d", false, "prevent further updates of the manifest")
	cmd.Flags().String("image-replacements", "", "path to image replacements file")
	must(cmd.Flags().MarkHidden("image-replacements"))
	cmd.Flags().Bool("skip-initializer", false, "skip injection of Contrast Initializer")
	cmd.Flags().Bool("skip-service-mesh", false, "skip injection of Contrast service mesh sidecar")
	cmd.Flags().Bool("inject-image-store", false, "inject an ephemeral storage device to pull images onto instead of into memory")
	cmd.Flags().Bool("insecure-enable-debug-shell-access", false, "enable the debug shell service in the pod CVM to get access from container to guest VM")
	cmd.Flags().StringP("output", "o", "", "output file for generated YAML")
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

	paths, err := findYamlFiles(args)
	if err != nil {
		return err
	}

	extraFile, err := os.CreateTemp("", "contrast-generate-extra-*.yml")
	if err != nil {
		return fmt.Errorf("create temp file for configmaps/secrets: %w", err)
	}
	defer os.Remove(extraFile.Name())

	fileMap, coordinatorNamespace, err := extractTargets(paths, extraFile, log)
	closeErr := extraFile.Close()
	if err != nil {
		return fmt.Errorf("extracting targets: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("closing temp file for configmaps/secrets: %w", closeErr)
	}

	verifiers := verifier.AllVerifiersBeforeGenerate(cmd)
	if err := runVerifiers(fileMap, verifiers); err != nil {
		return err
	}

	usedPlatforms, err := runtimeClassesFromUnstructured(fileMap)
	if err != nil {
		return fmt.Errorf("determining platforms used in deployment: %w", err)
	}
	if flags.referenceValuesPlatform != platforms.Unknown {
		usedPlatforms.Add(flags.referenceValuesPlatform)
	}

	// generate a manifest by checking if a manifest exists and using that,
	// or otherwise using a default.
	var mnf *manifest.Manifest
	existingManifest, err := os.ReadFile(flags.manifestPath)
	if errors.Is(err, fs.ErrNotExist) {
		// Manifest does not exist, create a new one
		mnf, err = manifest.Default(usedPlatforms.Platforms())
		if err != nil {
			return fmt.Errorf("create default manifest: %w", err)
		}
		if flags.referenceValuePatches != nil {
			if err := mnf.ReferenceValues.Patch(flags.referenceValuePatches); err != nil {
				return fmt.Errorf("patching reference values: %w", err)
			}
			if flags.purgeReferenceValues {
				mnf.ReferenceValues.PurgeEmpty()
			}
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

	var runtimeHandler string
	if flags.referenceValuesPlatform == platforms.Unknown {
		// Due to the pre generate verifiers, this code path should only be reachable when all resources have an explicit runtime class set.
		// The contrast-cc-unknown runtimeClassName should thus never end up in a generated resource.
		runtimeHandler = "contrast-cc-unknown"
	} else {
		runtimeHandler, err = manifest.RuntimeHandler(flags.referenceValuesPlatform)
		if err != nil {
			return fmt.Errorf("get runtime handler: %w", err)
		}
	}
	if err := patchTargets(log, fileMap, flags.imageReplacementsFile, runtimeHandler, coordinatorNamespace, flags); err != nil {
		return fmt.Errorf("patch targets: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Patched targets")

	if err := generatePolicies(cmd.Context(), flags, fileMap, extraFile.Name(), log); err != nil {
		return fmt.Errorf("generate policies: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "✔️ Generated workload policy annotations")

	var initdataManipulators []func(id *initdata.Initdata) error
	if flags.insecureEnableDebugShell {
		fmt.Fprintln(cmd.OutOrStdout(), "⚠️ Insecure debug shell access enabled!")
		initdataManipulators = append(initdataManipulators, func(id *initdata.Initdata) error {
			id.Data["contrast.insecure-debug"] = "true"
			return nil
		})
	}
	if err := manipulateInitdata(fileMap, initdataManipulators...); err != nil {
		return fmt.Errorf("manipulate initdata: %w", err)
	}

	policies, err := policiesFromKubeResources(fileMap)
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
		for _, e := range ve.Unwrap() {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", e)
		}
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

	verifiers = verifier.AllVerifiersAfterGenerate()
	if err := runVerifiers(fileMap, verifiers); err != nil {
		return err
	}

	if err := writeOutputFiles(fileMap, flags.outputFile); err != nil {
		return fmt.Errorf("write output files: %w", err)
	}

	return nil
}

// mapCCWorkloads applies the given function to all workloads with the 'contrast-cc' runtime class.
// The callback receives an apply configuration together with the file path and index the unstructured object has in the file map.
// Changes to the apply configuration are not applied to the original unstructured object.
func mapCCWorkloads(fileMap map[string][]*unstructured.Unstructured, f func(res any, path string, idx int) (any, error)) error {
	for path, resources := range fileMap {
		for idx, r := range resources {
			applyConfig, err := kuberesource.UnstructuredToApplyConfiguration(r)
			if err != nil {
				continue
			}
			if !isCCWorkload(applyConfig) {
				continue
			}
			changed, err := f(applyConfig, path, idx)
			if err != nil {
				return err
			}
			resUnstructured, err := kuberesource.ResourcesToUnstructured([]any{changed})
			if err != nil {
				return fmt.Errorf("convert patched resource to unstructured: %w", err)
			} else if len(resUnstructured) != 1 {
				return fmt.Errorf("expected 1 unstructured object, got %d", len(resUnstructured))
			}
			fileMap[path][idx] = resUnstructured[0]
		}
	}
	return nil
}

func isCCWorkload(resource any) (ret bool) {
	kuberesource.MapPodSpec(resource, func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		if spec != nil && spec.RuntimeClassName != nil && strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") {
			ret = true
		}
		return spec
	})
	return ret
}

func isCoordinator(resource any) bool {
	r, ok := resource.(*applyappsv1.StatefulSetApplyConfiguration)
	if ok &&
		r.Spec != nil &&
		r.Spec.Template != nil &&
		r.Spec.Template.ObjectMetaApplyConfiguration != nil &&
		r.Spec.Template.Annotations != nil &&
		r.Spec.Template.Annotations[contrastRoleAnnotationKey] == string(manifest.RoleCoordinator) {
		return true
	}
	return false
}

func runVerifiers(fileMap map[string][]*unstructured.Unstructured, verifiers []verifier.Verifier) error {
	var findings error
	for _, v := range verifiers {
		_ = mapCCWorkloads(fileMap, func(res any, path string, idx int) (any, error) {
			if err := v.Verify(res); err != nil {
				findings = errors.Join(findings, fmt.Errorf("failed to verify resource %q in file %q: %w", fileMap[path][idx].GetName(), path, err))
			}
			return res, nil
		})
	}
	if findings != nil {
		return findings
	}
	return nil
}

func findYamlFiles(args []string) ([]string, error) {
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
			return nil, fmt.Errorf("walk %s: %w", path, err)
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no .yml/.yaml files found")
	}

	return paths, nil
}

func extractTargets(paths []string, configFile io.Writer, logger *slog.Logger) (map[string][]*unstructured.Unstructured, string, error) {
	var extraResources []*unstructured.Unstructured
	fileMap := make(map[string][]*unstructured.Unstructured)
	var coordinatorNamespace string

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("Could not read file", "path", path, "err", err)
			continue
		}
		objects, err := kuberesource.UnmarshalUnstructuredK8SResource(data)
		if err != nil {
			logger.Warn("Could not parse file into Kubernetes resources", "path", path, "err", err)
			continue
		}
		containsCC := false
		for _, object := range objects {
			if object.GetKind() == "ConfigMap" || object.GetKind() == "Secret" {
				extraResources = append(extraResources, object)
			}
			fileMap[path] = append(fileMap[path], object)
			applyConfig, err := kuberesource.UnstructuredToApplyConfiguration(object)
			if err != nil {
				logger.Warn("Could not convert resource into ApplyConfiguration", "path", path, "err", err)
			} else if isCCWorkload(applyConfig) {
				containsCC = true
				if isCoordinator(applyConfig) {
					r, ok := applyConfig.(*applyappsv1.StatefulSetApplyConfiguration)
					if ok && r.ObjectMetaApplyConfiguration != nil && r.Namespace != nil {
						coordinatorNamespace = *r.Namespace
					}
				}
			}
		}
		if !containsCC {
			delete(fileMap, path)
		}
	}
	if len(fileMap) == 0 {
		return nil, "", fmt.Errorf("no .yml/.yaml files with 'contrast-cc' runtime found")
	}

	extraData, err := kuberesource.EncodeUnstructured(extraResources)
	if err != nil {
		return nil, "", fmt.Errorf("encoding configmaps/secrets: %w", err)
	}
	if _, err := configFile.Write(extraData); err != nil {
		return nil, "", fmt.Errorf("writing configmaps/secrets to temp file: %w", err)
	}
	return fileMap, coordinatorNamespace, nil
}

func generatePolicies(ctx context.Context, flags *generateFlags, fileMap map[string][]*unstructured.Unstructured, extraPath string, logger *slog.Logger) error {
	cfg := genpolicy.NewConfig()
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

	return mapCCWorkloads(fileMap, func(res any, path string, idx int) (any, error) {
		initdataAnno, err := runner.Run(ctx, res, extraPath, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to generate policy for %q in %q: %w", fileMap[path][idx].GetName(), path, err)
		}
		var retError error
		res = kuberesource.MapPodSpecWithMeta(res, func(
			meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration,
		) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
			if meta == nil {
				meta = &applymetav1.ObjectMetaApplyConfiguration{}
			}
			if meta.Annotations == nil {
				meta.Annotations = make(map[string]string)
			}
			meta.Annotations[initdata.InitdataAnnotationKey] = initdataAnno
			return meta, spec
		})
		return res, retError
	})
}

func patchTargets(logger *slog.Logger, fileMap map[string][]*unstructured.Unstructured, imageReplacementsFile, runtimeHandler, coordinatorNamespace string, flags *generateFlags) error {
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
	return mapCCWorkloads(fileMap, func(res any, _ string, _ int) (any, error) {
		if flags.insecureEnableDebugShell {
			if _, err := kuberesource.AddDebugShell(res, kuberesource.DebugShell()); err != nil {
				return nil, fmt.Errorf("injecting debug shell container: %w", err)
			}
		}
		if !flags.skipInitializer {
			if err := injectInitializer(res, coordinatorNamespace); err != nil {
				return nil, fmt.Errorf("injecting Initializer: %w", err)
			}
		}
		if !flags.skipServiceMesh {
			if err := injectServiceMesh(res); err != nil {
				return nil, fmt.Errorf("injecting Service Mesh: %w", err)
			}
		}
		if flags.injectImageStore {
			kuberesource.AddImageStore([]any{res})
		}

		kuberesource.PatchImages([]any{res}, replacements)

		replaceRuntimeClassName := patchRuntimeClassName(logger, runtimeHandler)
		kuberesource.MapPodSpec(res, replaceRuntimeClassName)

		if err := patchIDBlockAnnotation(res); err != nil {
			return nil, fmt.Errorf("injecting ID block annotations: %w", err)
		}

		return res, nil
	})
}

func injectInitializer(resource any, coordinatorNamespace string) error {
	if isCoordinator(resource) {
		return nil
	}
	if coordinatorNamespace == "" {
		coordinatorNamespace = "default"
	}
	coordinatorHost := fmt.Sprintf("coordinator-ready.%s", coordinatorNamespace)
	if _, err := kuberesource.AddInitializer(resource, kuberesource.Initializer(coordinatorHost)); err != nil {
		return err
	}
	return nil
}

func injectServiceMesh(resource any) error {
	if isCoordinator(resource) {
		return nil
	}
	if _, err := kuberesource.AddServiceMesh(resource, kuberesource.ServiceMeshProxy()); err != nil {
		return err
	}
	return nil
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

func writeOutputFiles(fileMap map[string][]*unstructured.Unstructured, outputFile string) error {
	var filesToWrite map[string][]*unstructured.Unstructured
	if outputFile != "" {
		var outputResources []*unstructured.Unstructured
		for _, resources := range fileMap {
			outputResources = append(outputResources, resources...)
		}
		filesToWrite = map[string][]*unstructured.Unstructured{
			outputFile: outputResources,
		}
	} else {
		filesToWrite = fileMap
	}
	for path, resources := range filesToWrite {
		data, err := kuberesource.EncodeUnstructured(resources)
		if err != nil {
			return fmt.Errorf("encoding resources: %w", err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("writing resource to %s: %w", path, err)
		}
	}
	return nil
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

func runtimeClassesFromUnstructured(fileMap map[string][]*unstructured.Unstructured) (kuberesource.PlatformCollection, error) {
	var res []any
	for _, resources := range fileMap {
		for _, r := range resources {
			applyConfig, err := kuberesource.UnstructuredToApplyConfiguration(r)
			if err != nil {
				return nil, err
			}
			res = append(res, applyConfig)
		}
	}
	runtimeClasses, err := kuberesource.CollectRuntimeClasses(res)
	if err != nil {
		return nil, err
	}
	return runtimeClasses, nil
}

func patchRuntimeClassName(logger *slog.Logger, defaultRuntimeHandler string) func(*applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
	return func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		if spec == nil || spec.RuntimeClassName == nil {
			return spec
		}
		if *spec.RuntimeClassName == "kata-cc-isolation" || *spec.RuntimeClassName == "contrast-cc" {
			spec.RuntimeClassName = &defaultRuntimeHandler
			return spec
		}
		if !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc-") {
			return spec
		}
		overridePlatform, err := platforms.FromRuntimeClassString(*spec.RuntimeClassName)
		spec.RuntimeClassName = &defaultRuntimeHandler
		if err != nil {
			logger.Error("could not determine platform for runtime class", "runtime-class-name", *spec.RuntimeClassName, "err", err)
			return spec
		}
		overrideRuntimeHandler, err := manifest.RuntimeHandler(overridePlatform)
		if err != nil {
			logger.Error("could not get runtime handler for platform", "platform", overridePlatform, "err", err)
			return spec
		}
		spec.RuntimeClassName = &overrideRuntimeHandler
		return spec
	}
}

func patchIDBlockAnnotation(res any) error {
	// runtime -> cpu_count -> product_line -> ID block
	var snpIDBlocks map[string]map[string]map[string]SnpIDBlock
	if err := json.Unmarshal(SNPIDBlocks, &snpIDBlocks); err != nil {
		return fmt.Errorf("unmarshal SNP ID blocks: %w", err)
	}

	var mapErr error
	mapFunc := func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
		if spec == nil || spec.RuntimeClassName == nil {
			return meta, spec
		}

		targetPlatform, err := platforms.FromRuntimeClassString(*spec.RuntimeClassName)
		if err != nil {
			mapErr = fmt.Errorf("determining platform from runtime class name %s: %w", *spec.RuntimeClassName, err)
			return meta, spec
		}
		if !platforms.IsSNP(targetPlatform) {
			return meta, spec
		}

		var regularContainersCPU int64
		for _, container := range spec.Containers {
			regularContainersCPU += getNeededCPUCount(container.Resources)
		}
		var initContainersCPU int64
		for _, container := range spec.InitContainers {
			cpuCount := getNeededCPUCount(container.Resources)
			initContainersCPU += cpuCount
			// Sidecar containers remain running alongside the actual application, consuming CPU resources
			if container.RestartPolicy != nil && *container.RestartPolicy == corev1.ContainerRestartPolicyAlways {
				regularContainersCPU += cpuCount
			}
		}
		podLevelCPU := getNeededCPUCount(spec.Resources)

		// Convert milliCPUs to number of CPUs (rounding up), and add 1 for hypervisor overhead
		totalMilliCPUs := max(regularContainersCPU, initContainersCPU, podLevelCPU)
		cpuCount := strconv.FormatInt((totalMilliCPUs+999)/1000+1, 10)

		platform := strings.ToLower(targetPlatform.String())

		// Ensure we pre-calculated the required blocks
		if snpIDBlocks[platform] == nil || snpIDBlocks[platform][amdCPUGenerationGenoa] == nil || snpIDBlocks[platform][amdCPUGenerationMilan] == nil ||
			snpIDBlocks[platform][amdCPUGenerationGenoa][cpuCount].IDBlock == "" || snpIDBlocks[platform][amdCPUGenerationMilan][cpuCount].IDBlock == "" {
			mapErr = fmt.Errorf("missing ID block configuration for runtime %s with %s CPUs", platform, cpuCount)
			return meta, spec
		}

		if meta.Annotations == nil {
			meta.Annotations = make(map[string]string, 3)
		}
		meta.Annotations[idBlockAnnotation+platform] = snpIDBlocks[platform][amdCPUGenerationGenoa][cpuCount].IDBlock
		meta.Annotations[idBlockAnnotation+platform] = snpIDBlocks[platform][amdCPUGenerationMilan][cpuCount].IDBlock
		meta.Annotations[hypervisorCPUCountAnnotation] = cpuCount

		return meta, spec
	}

	kuberesource.MapPodSpecWithMeta(res, mapFunc)
	return mapErr
}

func getNeededCPUCount(resources *applycorev1.ResourceRequirementsApplyConfiguration) int64 {
	if resources == nil {
		return 0
	}
	var requests, limits int64
	if resources.Requests != nil {
		requests = resources.Requests.Cpu().MilliValue()
	}
	if resources.Limits != nil {
		limits = resources.Limits.Cpu().MilliValue()
	}
	return max(requests, limits)
}

type generateFlags struct {
	policyPath               string
	settingsPath             string
	manifestPath             string
	genpolicyCachePath       string
	referenceValuesPlatform  platforms.Platform
	referenceValuePatches    manifest.ReferenceValuePatches
	purgeReferenceValues     bool
	workloadOwnerKeys        []string
	seedshareOwnerKeys       []string
	disableUpdates           bool
	workspaceDir             string
	imageReplacementsFile    string
	skipInitializer          bool
	skipServiceMesh          bool
	injectImageStore         bool
	insecureEnableDebugShell bool
	outputFile               string
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
	referenceValuesPlatform, err := platforms.FromStringOrEmpty(referenceValues)
	if err != nil {
		return nil, fmt.Errorf("invalid reference-values platform: %w", err)
	}
	referenceValuePatchFiles, err := cmd.Flags().GetStringArray("reference-value-patches")
	if err != nil {
		return nil, err
	}
	referenceValuePatches, err := manifest.PatchesFromFiles(referenceValuePatchFiles)
	if err != nil {
		return nil, err
	}
	purgeReferenceValues, err := cmd.Flags().GetBool("purge-empty-reference-values")
	if err != nil {
		return nil, err
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
	injectImageStore, err := cmd.Flags().GetBool("inject-image-store")
	if err != nil {
		return nil, err
	}
	insecureEnableDebugShell, err := cmd.Flags().GetBool("insecure-enable-debug-shell-access")
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
		policyPath:               policyPath,
		settingsPath:             settingsPath,
		genpolicyCachePath:       genpolicyCachePath,
		manifestPath:             manifestPath,
		referenceValuesPlatform:  referenceValuesPlatform,
		referenceValuePatches:    referenceValuePatches,
		purgeReferenceValues:     purgeReferenceValues,
		workloadOwnerKeys:        workloadOwnerKeys,
		seedshareOwnerKeys:       seedshareOwnerKeys,
		disableUpdates:           disableUpdates,
		workspaceDir:             workspaceDir,
		imageReplacementsFile:    imageReplacementsFile,
		skipInitializer:          skipInitializer,
		skipServiceMesh:          skipServiceMesh,
		injectImageStore:         injectImageStore,
		insecureEnableDebugShell: insecureEnableDebugShell,
		outputFile:               outputFile,
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
