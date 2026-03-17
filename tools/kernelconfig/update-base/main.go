// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/tools/kernelconfig/internal/kconfig"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: kernel-config-update <kata-release-tarball-path> <git-root-path>")
	}
	tarball := os.Args[1]
	gitroot := os.Args[2]

	configs := []struct {
		filename        string
		upstreamPattern *regexp.Regexp
		isGPU           bool
	}{
		{
			filename:        "config",
			upstreamPattern: regexp.MustCompile(`config-\d+\.\d+\.\d+-\d+$`),
			isGPU:           false,
		},
		{
			filename:        "config-nvidia-gpu",
			upstreamPattern: regexp.MustCompile(`config-\d+\.\d+\.\d+-\d+-nvidia-gpu$`),
			isGPU:           true,
		},
	}

	for _, cfgInfo := range configs {
		configFile := filepath.Join(gitroot, "tools", "kernelconfig", "internal", "base", cfgInfo.filename)
		targetFile := filepath.Join(gitroot, "tools", "kernelconfig", "internal", "kconfig", "testdata", fmt.Sprintf("expected-%s", cfgInfo.filename))

		matches, err := filepath.Glob(filepath.Join(tarball, "opt", "kata", "share", "kata-containers", "config-*"))
		if err != nil {
			log.Fatalf("failed to glob upstream config: %v", err)
		}
		matches = slices.DeleteFunc(matches, func(p string) bool {
			base := filepath.Base(p)
			return !cfgInfo.upstreamPattern.MatchString(base)
		})
		if !cfgInfo.isGPU {
			matches = slices.DeleteFunc(matches, func(p string) bool {
				return strings.Contains(p, "-nvidia-gpu-")
			})
		}
		if len(matches) == 0 {
			log.Fatalf("no upstream config found for pattern %s", cfgInfo.upstreamPattern)
		}
		if len(matches) > 1 {
			log.Fatalf("more than one upstream config found for pattern %s: %s", cfgInfo.upstreamPattern, matches)
		}
		upstreamData, err := os.ReadFile(matches[0])
		if err != nil {
			log.Fatalf("failed to read upstream config: %v", err)
		}
		if err := os.WriteFile(configFile, upstreamData, 0o644); err != nil {
			log.Fatalf("failed to write new config: %v", err)
		}

		config, err := kconfig.OverrideConfig(upstreamData, cfgInfo.isGPU)
		if err != nil {
			log.Fatalf("failed to parse upstream config: %v", err)
		}
		if err := os.WriteFile(targetFile, config.Marshal(), 0o644); err != nil {
			log.Fatalf("failed to write target config: %v", err)
		}
	}
}
