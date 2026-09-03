// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/cli/genpolicy"
	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
)

var (
	//go:embed assets/pod.yml
	podYaml []byte
	//go:embed assets/genpolicy-settings-kata.json
	genpolicySettings []byte
	//go:embed assets/images.json
	imagesJSON []byte
)

type testImage struct {
	// Path to the image tarball or OCI layout directory.
	Path string `json:"path"`
	// ReplaceRef is the image reference with which the image should be replaced with.
	ReplaceRef string `json:"ref"`
}

func main() {
	cmd := &cobra.Command{
		Use:   "policy-test",
		Short: "policy-test",
		RunE:  execute,
	}

	cmd.SilenceUsage = true

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func execute(c *cobra.Command, _ []string) error {
	// Start a local registry server to serve the test images.
	srv := &http.Server{Handler: registry.New(registry.Logger(log.New(io.Discard, "", 0)))}
	lis, err := (&net.ListenConfig{}).Listen(c.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer lis.Close()
	go func() {
		if err := srv.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("registry server error: %v", err)
		}
	}()
	defer srv.Close()
	registryAddr := lis.Addr().String()

	// Push the test images to the local registry.
	var testImages map[string]testImage
	if err := json.Unmarshal(imagesJSON, &testImages); err != nil {
		return fmt.Errorf("unmarshal images.json: %w", err)
	}
	imageReplacements, err := setupRegistry(registryAddr, testImages)
	if err != nil {
		return fmt.Errorf("setup registry: %w", err)
	}

	imageReplacementsFile, err := os.CreateTemp("", "image-replacements-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(imageReplacementsFile.Name())
	for k, v := range imageReplacements {
		if _, err := fmt.Fprintf(imageReplacementsFile, "%s=%s\n", k, v); err != nil {
			return fmt.Errorf("write image replacements: %w", err)
		}
		podYaml = bytes.ReplaceAll(podYaml, []byte(k), []byte(v))
	}
	if err := imageReplacementsFile.Close(); err != nil {
		return fmt.Errorf("close image replacements file: %w", err)
	}

	workDir, err := os.MkdirTemp("", "contrast-policy-test-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	if err := os.WriteFile(filepath.Join(workDir, "pod.yml"), podYaml, 0o644); err != nil {
		return fmt.Errorf("write pod.yml: %w", err)
	}

	// Patch the pause image in genpolicy-settings.json to use the local registry.
	pauseImageRef, ok := testImages["pause"]
	if !ok {
		return fmt.Errorf("pause image not found in test images")
	}
	if !bytes.Contains(genpolicySettings, []byte(pauseImageRef.ReplaceRef)) {
		return fmt.Errorf("pause image reference %s not found in genpolicy-settings.json", pauseImageRef.ReplaceRef)
	}
	pauseImage, ok := imageReplacements[pauseImageRef.ReplaceRef]
	if !ok {
		return fmt.Errorf("pause image not found in image replacements")
	}
	genpolicySettings = bytes.ReplaceAll(genpolicySettings, []byte(pauseImageRef.ReplaceRef), []byte(pauseImage))
	if err := os.WriteFile(filepath.Join(workDir, "genpolicy-settings.json"), genpolicySettings, 0o644); err != nil {
		return fmt.Errorf("write genpolicy-settings.json: %w", err)
	}

	genpolicyConfig := genpolicy.NewConfig()
	if err := os.WriteFile(filepath.Join(workDir, "genpolicy-rules.rego"), genpolicyConfig.Rules, 0o644); err != nil {
		return fmt.Errorf("write genpolicy-rules.rego: %w", err)
	}
	genpolicyRunner, err := genpolicy.New(
		filepath.Join(workDir, "genpolicy-rules.rego"),
		filepath.Join(workDir, "genpolicy-settings.json"),
		filepath.Join(workDir, "layers-cache.json"),
		[]string{registryAddr},
		genpolicyConfig.Bin,
	)
	if err != nil {
		return fmt.Errorf("create genpolicy runner: %w", err)
	}

	policy, err := generatePolicy(c.Context(), genpolicyRunner, podYaml)
	if err != nil {
		return fmt.Errorf("generate policy: %w", err)
	}

	dataDir := "./policy-test/testdata/"
	dirs, err := os.ReadDir(dataDir)
	if err != nil {
		return fmt.Errorf("read test data dir: %w", err)
	}

	var numTestCases int
	var errs []error
	for _, file := range dirs {
		if file.IsDir() {
			continue
		}
		numTestCases++
		log.Printf("===== Running test case: %s", file.Name())
		fileData, err := os.ReadFile(filepath.Join(dataDir, file.Name()))
		if err != nil {
			errs = append(errs, fmt.Errorf("read test case file %s: %w", file.Name(), err))
			continue
		}
		var tc []TestCase
		if err := json.Unmarshal(fileData, &tc); err != nil {
			errs = append(errs, fmt.Errorf("unmarshal test case %s: %w", file.Name(), err))
			continue
		}

		p, err := NewOPAPolicy(policy)
		if err != nil {
			errs = append(errs, fmt.Errorf("create OPA policy for test case %s: %w", file.Name(), err))
			continue
		}
		for _, testCase := range tc {
			allowed, prints, err := p.AllowRequest(c.Context(), testCase)
			if err != nil {
				log.Printf("%s: policy evaluation error: %v\n", testCase.Kind, err)
				errs = append(errs, fmt.Errorf("test case %s failed: %w, logs:\n%v", file.Name(), err, prints))
				break
			}
			if allowed != testCase.Allowed {
				log.Printf("%s: policy evaluation mismatch, expected allowed=%v, got allowed=%v\n", testCase.Kind, testCase.Allowed, allowed)
				errs = append(errs, fmt.Errorf("test case %s failed: %v: expected allowed=%v, got allowed=%v, logs:\n%v", file.Name(), testCase.Kind, testCase.Allowed, allowed, prints))
				break
			}
			log.Printf("%s: allowed=%v\n", testCase.Kind, allowed)
		}
	}

	if numTestCases == 0 {
		return fmt.Errorf("no test cases found in %s", dataDir)
	}

	return errors.Join(errs...)
}

func setupRegistry(registryAddr string, testImages map[string]testImage) (map[string]string, error) {
	imageReplacements := make(map[string]string)
	for imgName, img := range testImages {
		ref, err := name.NewTag(registryAddr+"/"+imgName, name.Insecure)
		if err != nil {
			return nil, fmt.Errorf("parse image name %s: %w", imgName, err)
		}
		idx, err := layout.ImageIndexFromPath(img.Path)
		if err != nil {
			return nil, fmt.Errorf("load image index %s: %w", img.Path, err)
		}
		idxManifest, err := idx.IndexManifest()
		if err != nil {
			return nil, fmt.Errorf("get index manifest for image index %s: %w", img.Path, err)
		}
		if len(idxManifest.Manifests) != 1 {
			return nil, fmt.Errorf("expected exactly one manifest in image index %s, got %d", img.Path, len(idxManifest.Manifests))
		}
		digest := idxManifest.Manifests[0].Digest
		if err := remote.WriteIndex(ref, idx); err != nil {
			return nil, fmt.Errorf("write image index %s to registry: %w", imgName, err)
		}
		imageReplacements[img.ReplaceRef] = ref.String() + "@" + digest.String()
	}
	return imageReplacements, nil
}

func generatePolicy(ctx context.Context, runner *genpolicy.Runner, yaml []byte) (string, error) {
	applyConfigs, err := kuberesource.UnmarshalApplyConfigurations(yaml)
	if err != nil {
		return "", fmt.Errorf("unmarshal pod yaml: %w", err)
	}
	if len(applyConfigs) != 1 {
		return "", fmt.Errorf("expected exactly one resource, got %d", len(applyConfigs))
	}
	// TODO(davidweisse): Change extraPath when we actually need it.
	anno, _, err := runner.Run(ctx, applyConfigs[0], "/dev/null", false, slog.Default())
	if err != nil {
		return "", fmt.Errorf("run genpolicy: %w", err)
	}
	idRaw, err := initdata.DecodeKataAnnotation(anno)
	if err != nil {
		return "", fmt.Errorf("decoding initdata annotation: %w", err)
	}
	id, err := idRaw.Parse()
	if err != nil {
		return "", fmt.Errorf("parsing initdata: %w", err)
	}
	if policy, ok := id.Data["policy.rego"]; ok {
		return policy, nil
	}
	return "", fmt.Errorf("policy.rego not found in initdata")
}
