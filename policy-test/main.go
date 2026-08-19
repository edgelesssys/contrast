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
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/spf13/cobra"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

//go:embed assets/pod.yml
var podYaml []byte

func main() {
	cmd := &cobra.Command{
		Use:   "policy-test",
		Short: "policy-test",
		RunE:  execute,
	}
	cmd.Flags().String("image-replacements", "", "path to the image replacements file")

	cmd.SilenceUsage = true

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func execute(c *cobra.Command, _ []string) error {
	flags, err := parseFlags(c)
	if err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	workDir, err := os.MkdirTemp("", "contrast-policy-test-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	if err := os.WriteFile(filepath.Join(workDir, "pod.yml"), podYaml, 0o644); err != nil {
		return fmt.Errorf("write pod.yml: %w", err)
	}

	policy, err := generatePolicy(c.Context(), workDir, flags)
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

func generatePolicy(ctx context.Context, workDir string, flags *flags) (string, error) {
	generateCmd := cmd.NewGenerateCmd()
	generateCmd.Flags().String("workspace-dir", workDir, "")
	args := []string{
		"--workspace-dir=" + workDir,
		"--reference-values=Metal-QEMU-SNP",
		"--genpolicy-cache-path=" + filepath.Join(workDir, "layers-cache.json"),
		"--image-replacements=" + flags.imageReplacementsFile,
		"--output=" + filepath.Join(workDir, "out.yml"),
		filepath.Join(workDir, "pod.yml"),
	}
	generateCmd.SetArgs(args)
	generateCmd.SetOut(io.Discard)
	errBuf := &bytes.Buffer{}
	generateCmd.SetErr(errBuf)

	if err := generateCmd.ExecuteContext(ctx); err != nil {
		return "", fmt.Errorf("run generate: %w\n%s", err, errBuf.String())
	}

	output, err := os.ReadFile(filepath.Join(workDir, "out.yml"))
	if err != nil {
		return "", err
	}

	policy, err := extractPolicy(output)
	if err != nil {
		return "", fmt.Errorf("extract policy: %w", err)
	}
	return policy, nil
}

func extractPolicy(yaml []byte) (string, error) {
	applyConfigs, err := kuberesource.UnmarshalApplyConfigurations(yaml)
	if err != nil {
		return "", err
	}
	if len(applyConfigs) != 1 {
		return "", fmt.Errorf("expected exactly one resource, got %d", len(applyConfigs))
	}
	var policy string
	_, err = kuberesource.MapPodSpecWithMetaAndErrors(applyConfigs[0], func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration, error) {
		if meta == nil {
			return meta, spec, fmt.Errorf("pod spec is nil")
		}
		annotation := meta.Annotations[kuberesource.InitdataAnnotationKey]
		idRaw, err := initdata.DecodeKataAnnotation(annotation)
		if err != nil {
			return meta, spec, fmt.Errorf("decoding initdata annotation: %w", err)
		}
		id, err := idRaw.Parse()
		if err != nil {
			return meta, spec, fmt.Errorf("parsing initdata: %w", err)
		}
		var ok bool
		if policy, ok = id.Data["policy.rego"]; !ok {
			return meta, spec, fmt.Errorf("policy.rego not found in initdata")
		}
		return meta, spec, nil
	})
	if err != nil {
		return "", err
	}
	return policy, nil
}

type flags struct {
	imageReplacementsFile string
}

func parseFlags(cmd *cobra.Command) (*flags, error) {
	imageReplacementsFile, err := cmd.Flags().GetString("image-replacements")
	if err != nil {
		return nil, err
	}
	return &flags{
		imageReplacementsFile: imageReplacementsFile,
	}, nil
}
