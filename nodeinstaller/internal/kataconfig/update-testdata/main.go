// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Update the configs and testdata for all platforms when they change upstream in kata-containers.
// Should be invoked via `nix run .#scripts.update-kata-configurations`.
// This is also part of the `static` CI workflow through `nix run .#scripts.generate`.
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/kataconfig"
)

func main() {
	tarball := os.Args[1]
	gitroot := os.Args[2]

	platforms := map[platforms.Platform]struct {
		upstream string
		config   string
		testdata string
	}{
		platforms.AKSCloudHypervisorSNP: {
			upstream: "clh",
			config:   "clh-snp",
			testdata: "clh-snp",
		},
		platforms.MetalQEMUSNP: {
			upstream: "qemu-snp",
			config:   "qemu-snp",
			testdata: "qemu-snp",
		},
		platforms.MetalQEMUTDX: {
			upstream: "qemu-tdx",
			config:   "qemu-tdx",
			testdata: "qemu-tdx",
		},
		platforms.MetalQEMUSNPGPU: {
			upstream: "qemu-snp",
			config:   "qemu-snp",
			testdata: "qemu-snp-gpu",
		},
	}

	snpIDBlock := kataconfig.SnpIDBlock{
		IDAuth:  "PLACEHOLDER_ID_AUTH",
		IDBlock: "PLACEHOLDER_ID_BLOCK",
	}

	exit := 0
	for platform, platformConfig := range platforms {
		upstreamFile := filepath.Join(tarball, "opt", "kata", "share", "defaults", "kata-containers", fmt.Sprintf("configuration-%s.toml", platformConfig.upstream))
		configFile := filepath.Join(gitroot, "nodeinstaller", "internal", "kataconfig", fmt.Sprintf("configuration-%s.toml", platformConfig.config))
		testdataFile := filepath.Join(gitroot, "nodeinstaller", "internal", "kataconfig", "testdata", fmt.Sprintf("expected-configuration-%s.toml", platformConfig.testdata))

		upstream, err := os.ReadFile(upstreamFile)
		if err != nil {
			log.Fatalf("opening upstream file: %s", err)
		}

		config, err := os.ReadFile(configFile)
		if err != nil {
			log.Fatalf("opening config file: %s", err)
		}

		if bytes.Equal(upstream, config) {
			log.Printf("✔ No upstream changes for platform %s.", platform.String())
		} else {
			// Continuing anyway to prevent missing config updates due to the platformConfig.config file changing in iteration i,
			// and then being assessed as unchanged in iteration i+j (example: changes to qemu-snp also need updates in qemu-snp-gpu)
			log.Printf("⚠ Updating config for platform %s.", platform.String())
			exit = 1
		}

		if err := os.WriteFile(configFile, upstream, 0o644); err != nil {
			log.Fatalf("failed to write new config: %s", err)
		}

		cfg, err := kataconfig.KataRuntimeConfig("/", platform, "", snpIDBlock, false)
		if err != nil {
			log.Fatalf("failed to create config: %s", err)
		}

		cfgBytes, err := cfg.Marshal()
		if err != nil {
			log.Fatalf("failed to marshal config: %s", err)
		}

		if err := os.WriteFile(testdataFile, cfgBytes, 0o644); err != nil {
			log.Fatalf("failed to write testdata: %s", err)
		}
	}

	os.Exit(exit)
}
