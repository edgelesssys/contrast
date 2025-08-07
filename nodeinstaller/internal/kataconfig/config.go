// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kataconfig

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/pelletier/go-toml/v2"
)

var (
	// kataCLHSNPBaseConfig is the configuration file for the Kata runtime on AKS SEV-SNP
	// with Cloud-Hypervisor.
	//
	//go:embed configuration-clh-snp.toml
	kataCLHSNPBaseConfig string

	// kataBareMetalQEMUTDXBaseConfig is the configuration file for the Kata runtime on bare-metal TDX
	// with QEMU.
	//
	//go:embed configuration-qemu-tdx.toml
	kataBareMetalQEMUTDXBaseConfig string

	// kataBareMetalQEMUSNPBaseConfig is the configuration file for the Kata runtime on bare-metal SNP
	// with QEMU.
	//
	//go:embed configuration-qemu-snp.toml
	kataBareMetalQEMUSNPBaseConfig string

	//go:embed snp-id-blocks.json
	snpIDBlocks string

	// RuntimeNamePlaceholder is the placeholder for the per-runtime path (i.e. /opt/edgeless/contrast-cc...) in the target file paths.
	RuntimeNamePlaceholder = "@@runtimeName@@"
)

// KataRuntimeConfig returns the Kata runtime configuration.
func KataRuntimeConfig(
	baseDir string,
	platform platforms.Platform,
	qemuExtraKernelParams string,
	snpIDBlock SnpIDBlock,
	debug bool,
) (*Config, error) {
	var config Config
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		if err := toml.Unmarshal([]byte(kataCLHSNPBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}
		// Use the resources installed by Contrast node-installer.
		config.Hypervisor["clh"]["path"] = filepath.Join(baseDir, "bin", "cloud-hypervisor-snp")
		config.Hypervisor["clh"]["igvm"] = filepath.Join(baseDir, "share", "kata-containers-igvm.img")
		config.Hypervisor["clh"]["kernel"] = nil       // Already part of IGVM.
		config.Hypervisor["clh"]["kernel_params"] = "" // Already part of IGVM.
		config.Hypervisor["clh"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
		config.Hypervisor["clh"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "bin", "cloud-hypervisor-snp")}
		// Fix and align guest memory calculation.
		config.Hypervisor["clh"]["default_memory"] = platforms.DefaultMemoryInMegaBytes(platform)
		config.Runtime["sandbox_cgroup_only"] = true
		// Conditionally enable debug mode.
		config.Hypervisor["clh"]["enable_debug"] = debug
		// Increase dial timeout and accept slower guest startup times.
		config.Agent["kata"]["dial_timeout"] = 90
		// Disable all annotations, as we don't support these. Some will mess up measurements,
		// others bypass things you can archive via correct resource declaration anyway.
		config.Hypervisor["clh"]["enable_annotations"] = []string{}

		// Upstream clh config for SNP doesn't exist, configure it here.
		// TODO(katexochen): Add a clh-snp configuration upstream.
		config.Hypervisor["clh"]["confidential_guest"] = true
		config.Hypervisor["clh"]["sev_snp_guest"] = true
		config.Hypervisor["clh"]["shared_fs"] = "none"
		config.Hypervisor["clh"]["snp_guest_policy"] = 196608
		config.Runtime["static_sandbox_resource_mgmt"] = true
	case platforms.MetalQEMUTDX:
		if err := toml.Unmarshal([]byte(kataBareMetalQEMUTDXBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}
		// Use the resources installed by Contrast node-installer.
		config.Hypervisor["qemu"]["path"] = filepath.Join(baseDir, "tdx", "bin", "qemu-system-x86_64")
		config.Hypervisor["qemu"]["firmware"] = filepath.Join(baseDir, "tdx", "share", "OVMF.fd")
		config.Hypervisor["qemu"]["initrd"] = filepath.Join(baseDir, "share", "kata-initrd.zst")
		config.Hypervisor["qemu"]["kernel"] = filepath.Join(baseDir, "share", "kata-kernel")
		config.Hypervisor["qemu"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
		config.Hypervisor["qemu"]["rootfs_type"] = "erofs"
		config.Hypervisor["qemu"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "tdx", "bin", "qemu-system-x86_64")}
		// Fix and align guest memory calculation.
		config.Hypervisor["qemu"]["default_memory"] = platforms.DefaultMemoryInMegaBytes(platform)
		config.Runtime["sandbox_cgroup_only"] = true
		// Force container image gust pull so we don't have to use nydus-snapshotter.
		config.Runtime["experimental_force_guest_pull"] = true
		// Replace the kernel params entirely (and don't append) since that's
		// also what we do when calculating the launch measurement.
		config.Hypervisor["qemu"]["kernel_params"] = qemuExtraKernelParams
		// Conditionally enable debug mode.
		config.Hypervisor["qemu"]["enable_debug"] = debug
		// Disable all annotations, as we don't support these. Some will mess up measurements,
		// others bypass things you can archive via correct resource declaration anyway.
		config.Hypervisor["qemu"]["enable_annotations"] = []string{}

		// TODO: Check again why we need this and how we can avoid it.
		config.Hypervisor["qemu"]["block_device_aio"] = "threads"
	case platforms.MetalQEMUSNP, platforms.MetalQEMUSNPGPU:
		if err := toml.Unmarshal([]byte(kataBareMetalQEMUSNPBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}
		// Use the resources installed by Contrast node-installer.
		config.Hypervisor["qemu"]["path"] = filepath.Join(baseDir, "snp", "bin", "qemu-system-x86_64")
		config.Hypervisor["qemu"]["firmware"] = filepath.Join(baseDir, "snp", "share", "OVMF.fd")
		config.Hypervisor["qemu"]["initrd"] = filepath.Join(baseDir, "share", "kata-initrd.zst")
		config.Hypervisor["qemu"]["kernel"] = filepath.Join(baseDir, "share", "kata-kernel")
		config.Hypervisor["qemu"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
		config.Hypervisor["qemu"]["rootfs_type"] = "erofs"
		config.Hypervisor["qemu"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "snp", "bin", "qemu-system-x86_64")}
		// Fix and align guest memory calculation.
		config.Hypervisor["qemu"]["default_memory"] = platforms.DefaultMemoryInMegaBytes(platform)
		config.Runtime["sandbox_cgroup_only"] = true
		// Force container image gust pull so we don't have to use nydus-snapshotter.
		config.Runtime["experimental_force_guest_pull"] = true
		// Replace the kernel params entirely (and don't append) since that's
		// also what we do when calculating the launch measurement.
		config.Hypervisor["qemu"]["kernel_params"] = qemuExtraKernelParams
		// TODO: Check again why we need this and how we can avoid it.
		config.Hypervisor["qemu"]["block_device_aio"] = "threads"
		// Add SNP ID block to protect against migration attacks.
		config.Hypervisor["qemu"]["snp_id_block"] = snpIDBlock.IDBlock
		config.Hypervisor["qemu"]["snp_id_auth"] = snpIDBlock.IDAuth
		// Conditionally enable debug mode.
		config.Hypervisor["qemu"]["enable_debug"] = debug
		// Disable all annotations, as we don't support these. Some will mess up measurements,
		// others bypass things you can archive via correct resource declaration anyway.
		config.Hypervisor["qemu"]["enable_annotations"] = []string{}

		// GPU-specific settings
		if platforms.IsGPU(platform) {
			config.Hypervisor["qemu"]["guest_hook_path"] = "/usr/share/oci/hooks"
			config.Hypervisor["qemu"]["cold_plug_vfio"] = "root-port"
			// GPU images tend to be larger, so give a better default timeout that
			// allows for pulling those.
			config.Agent["kata"]["dial_timeout"] = 600
			config.Runtime["create_container_timeout"] = 600
		}
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
	if debug {
		config.Agent["kata"]["enable_debug"] = true
		config.Agent["kata"]["debug_console_enabled"] = true
		config.Runtime["enable_debug"] = true
	}
	return &config, nil
}

// SnpIDBlock represents the SNP ID block and ID auth used for SEV-SNP guests.
type SnpIDBlock struct {
	IDBlock string `json:"idBlock"`
	IDAuth  string `json:"idAuth"`
}

// platform -> product -> snpIDBlock.
type snpIDBlockMap map[string]map[string]SnpIDBlock

// SnpIDBlockForPlatform returns the embedded SNP ID block and ID auth for the given platform and product.
func SnpIDBlockForPlatform(platform platforms.Platform, productName sevsnp.SevProduct_SevProductName) (SnpIDBlock, error) {
	blocks := make(snpIDBlockMap)
	if err := json.Unmarshal([]byte(snpIDBlocks), &blocks); err != nil {
		return SnpIDBlock{}, fmt.Errorf("unmarshaling embedded SNP ID blocks: %w", err)
	}
	blockForPlatform, ok := blocks[strings.ToLower(platform.String())]
	if !ok {
		return SnpIDBlock{}, fmt.Errorf("no SNP ID block found for platform %s", platform)
	}
	productLine := kds.ProductLine(&sevsnp.SevProduct{Name: productName})
	block, ok := blockForPlatform[productLine]
	if !ok {
		return SnpIDBlock{}, fmt.Errorf("no SNP ID block found for product %s", productLine)
	}
	return block, nil
}

// Config is the configuration for the Kata runtime.
// Source: https://github.com/kata-containers/kata-containers/blob/4029d154ba0c26fcf4a8f9371275f802e3ef522c/src/runtime/pkg/katautils/Config.go
// This is a simplified version of the actual configuration.
type Config struct {
	Hypervisor map[string]hypervisorConfig
	Agent      map[string]agentConfig
	Image      imageConfig
	Factory    factoryConfig
	Runtime    runtimeConfig
}

// Marshal encodes the configuration as TOML.
func (k *Config) Marshal() ([]byte, error) {
	return toml.Marshal(k)
}

// imageConfig is the configuration for the image.
type imageConfig map[string]any

// factoryConfig is the configuration for the factory.
type factoryConfig map[string]any

// hypervisorConfig is the configuration for the hypervisor.
type hypervisorConfig map[string]any

// runtimeConfig is the configuration for the Kata runtime.
type runtimeConfig map[string]any

// agentConfig is the configuration for the agent.
type agentConfig map[string]any
