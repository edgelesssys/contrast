// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/edgelesssys/contrast/cli/genpolicy"
	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"
)

var (
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

func TestPolicy(t *testing.T) {
	require := require.New(t)

	// Start a local registry server to serve the test images.
	errCh := make(chan error)
	srv := &http.Server{Handler: registry.New(registry.Logger(log.New(io.Discard, "", 0)))}
	lis, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(err)
	registryAddr := lis.Addr().String()

	t.Cleanup(func() {
		require.NoError(srv.Close())
		err := <-errCh
		require.ErrorIs(err, http.ErrServerClosed)
	})

	go func() {
		errCh <- srv.Serve(lis)
	}()

	// Push the test images to the local registry.
	var testImages map[string]testImage
	require.NoError(json.Unmarshal(imagesJSON, &testImages))
	imageReplacements, err := setupRegistry(registryAddr, testImages)
	require.NoError(err)

	workDir := t.TempDir()

	// Patch the pause image in genpolicy-settings.json to use the local registry.
	pauseImageRef, ok := testImages["pause"]
	require.True(ok, "pause image not found in test images")
	require.Contains(string(genpolicySettings), pauseImageRef.ReplaceRef, "pause image reference %s not found in genpolicy-settings.json", pauseImageRef.ReplaceRef)
	pauseImage, ok := imageReplacements[pauseImageRef.ReplaceRef]
	require.True(ok, "pause image not found in image replacements")
	genpolicySettings = bytes.ReplaceAll(genpolicySettings, []byte(pauseImageRef.ReplaceRef), []byte(pauseImage))
	require.NoError(os.WriteFile(filepath.Join(workDir, "genpolicy-settings.json"), genpolicySettings, 0o644))

	genpolicyConfig := genpolicy.NewConfig()
	require.NoError(os.WriteFile(filepath.Join(workDir, "genpolicy-rules.rego"), genpolicyConfig.Rules, 0o644))
	genpolicyRunner, err := genpolicy.New(
		filepath.Join(workDir, "genpolicy-rules.rego"),
		filepath.Join(workDir, "genpolicy-settings.json"),
		filepath.Join(workDir, "layers-cache.json"),
		[]string{registryAddr},
		genpolicyConfig.Bin,
	)
	require.NoError(err)

	dataDir := "./policy-test/testdata/"
	dirs, err := os.ReadDir(dataDir)
	require.NoError(err)

	var numTestCases int
	for _, subdir := range dirs {
		if !subdir.IsDir() {
			continue
		}
		numTestCases++
		runTest(t, genpolicyRunner, imageReplacements, workDir, dataDir, subdir.Name())
	}

	require.Positive(numTestCases, "no test cases found in %s", dataDir)
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

func runTest(t *testing.T, runner *genpolicy.Runner, imageReplacements map[string]string, workDir, dataDir, name string) {
	t.Run(name, func(t *testing.T) {
		req := require.New(t)
		yaml, err := os.ReadFile(filepath.Join(dataDir, name, "resource.yml"))
		req.NoError(err)
		for k, v := range imageReplacements {
			yaml = bytes.ReplaceAll(yaml, []byte(k), []byte(v))
		}
		req.NoError(os.WriteFile(filepath.Join(workDir, "resource.yml"), yaml, 0o644))
		policy, err := generatePolicy(t.Context(), runner, yaml)
		req.NoError(err)

		baseData, err := os.ReadFile(filepath.Join(dataDir, name, "base.json"))
		req.NoError(err)
		files, err := os.ReadDir(filepath.Join(dataDir, name))
		req.NoError(err)

		require.True(t, t.Run("base.json", func(t *testing.T) {
			runTestCase(t, policy, baseData)
		}), "base.json test case needs to succeed for other subtests to run")

		for _, file := range files {
			if file.Name() == "base.json" || filepath.Ext(file.Name()) != ".json" {
				continue
			}
			t.Run(file.Name(), func(t *testing.T) {
				require := require.New(t)
				patchData, err := os.ReadFile(filepath.Join(dataDir, name, file.Name()))
				require.NoError(err)
				patch, err := jsonpatch.DecodePatch(patchData)
				require.NoError(err)
				testData, err := patch.Apply(baseData)
				require.NoError(err)
				runTestCase(t, policy, testData)
			})
		}
	})
}

func runTestCase(t *testing.T, policy string, data []byte) {
	require := require.New(t)
	var tc []TestCase
	require.NoError(json.Unmarshal(data, &tc))
	p, err := NewOPAPolicy(policy)
	require.NoError(err)
	for _, testCase := range tc {
		allowed, prints, err := p.AllowRequest(t.Context(), testCase)
		require.NoError(err, prints)
		t.Logf("%s: %v", testCase.Kind, allowed)
		require.Equal(testCase.Allowed, allowed, "policy mismatch for %s:\n%s", testCase.Kind, prints)
	}
}
