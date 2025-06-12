// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package constants

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/config"
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

	// containerdBaseConfig is the base configuration file for containerd
	//
	//go:embed containerd-config.toml
	containerdBaseConfig string

	//go:embed snp-id-blocks.json
	snpIDBlocks string

	// RuntimeNamePlaceholder is the placeholder for the per-runtime path (i.e. /opt/edgeless/contrast-cc...) in the target file paths.
	RuntimeNamePlaceholder = "@@runtimeName@@"
)

// CRIFQDN is the fully qualified domain name of the CRI service, which depends on the containerd config version.
func CRIFQDN(v int) string {
	switch v {
	case 3:
		return "io.containerd.cri.v1.runtime"
	default:
		return "io.containerd.grpc.v1.cri"
	}
}

// ImagesFQDN is the fully qualified domain name of the images plugin, which was factored out of the CRI plugin in containerd v2.
func ImagesFQDN(v int) string {
	switch v {
	case 3:
		return "io.containerd.cri.v1.images"
	default:
		return "io.containerd.grpc.v1.cri"
	}
}

// KataRuntimeConfig returns the Kata runtime configuration.
func KataRuntimeConfig(
	baseDir string,
	platform platforms.Platform,
	qemuExtraKernelParams string,
	snpIDBlock SnpIDBlock,
	debug bool,
) (*config.KataRuntimeConfig, error) {
	var config config.KataRuntimeConfig
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		if err := toml.Unmarshal([]byte(kataCLHSNPBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}
		config.Hypervisor["clh"]["path"] = filepath.Join(baseDir, "bin", "cloud-hypervisor-snp")
		config.Hypervisor["clh"]["igvm"] = filepath.Join(baseDir, "share", "kata-containers-igvm.img")
		config.Hypervisor["clh"]["kernel"] = nil
		config.Hypervisor["clh"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
		config.Hypervisor["clh"]["default_memory"] = platforms.DefaultMemoryInMegaBytes(platform)
		config.Hypervisor["clh"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "bin", "cloud-hypervisor-snp")}
		config.Hypervisor["clh"]["enable_debug"] = debug
		config.Hypervisor["clh"]["confidential_guest"] = true
		config.Hypervisor["clh"]["sev_snp_guest"] = true
		config.Hypervisor["clh"]["shared_fs"] = "none"
		config.Hypervisor["clh"]["snp_guest_policy"] = 196608

		config.Agent["kata"]["dial_timeout"] = 90

		config.Image = make(map[string]any)
		config.Image["service_offload"] = false

		config.Runtime["sandbox_cgroup_only"] = true
		config.Runtime["static_sandbox_resource_mgmt"] = true
		config.Runtime["static_sandbox_default_workload_mem"] = 1792
	case platforms.MetalQEMUTDX, platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
		if err := toml.Unmarshal([]byte(kataBareMetalQEMUTDXBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}
		config.Runtime["force_guest_pull"] = true
		config.Hypervisor["qemu"]["path"] = filepath.Join(baseDir, "tdx", "bin", "qemu-system-x86_64")
		config.Hypervisor["qemu"]["firmware"] = filepath.Join(baseDir, "tdx", "share", "OVMF.fd")
		config.Hypervisor["qemu"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
		config.Hypervisor["qemu"]["default_memory"] = platforms.DefaultMemoryInMegaBytes(platform)
		config.Hypervisor["qemu"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "tdx", "bin", "qemu-system-x86_64")}
		config.Hypervisor["qemu"]["block_device_aio"] = "threads"
		config.Hypervisor["qemu"]["rootfs_type"] = "erofs"
		config.Hypervisor["qemu"]["shared_fs"] = "none"
		config.Hypervisor["qemu"]["initrd"] = filepath.Join(baseDir, "share", "kata-initrd.zst")
		config.Hypervisor["qemu"]["kernel"] = filepath.Join(baseDir, "share", "kata-kernel")
		// Replace the kernel params entirely (and don't append) since that's
		// also what we do when calculating the launch measurement.
		config.Hypervisor["qemu"]["kernel_params"] = qemuExtraKernelParams
		if debug {
			config.Hypervisor["qemu"]["enable_debug"] = true
		}
	case platforms.MetalQEMUSNP, platforms.K3sQEMUSNP, platforms.K3sQEMUSNPGPU,
		platforms.MetalQEMUSNPGPU:
		if err := toml.Unmarshal([]byte(kataBareMetalQEMUSNPBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}
		config.Runtime["force_guest_pull"] = true
		config.Hypervisor["qemu"]["path"] = filepath.Join(baseDir, "snp", "bin", "qemu-system-x86_64")
		config.Hypervisor["qemu"]["firmware"] = filepath.Join(baseDir, "snp", "share", "OVMF.fd")
		config.Hypervisor["qemu"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
		config.Hypervisor["qemu"]["default_memory"] = platforms.DefaultMemoryInMegaBytes(platform)
		config.Hypervisor["qemu"]["block_device_aio"] = "threads"
		config.Hypervisor["qemu"]["shared_fs"] = "none"
		config.Hypervisor["qemu"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "snp", "bin", "qemu-system-x86_64")}
		config.Hypervisor["qemu"]["rootfs_type"] = "erofs"
		config.Hypervisor["qemu"]["initrd"] = filepath.Join(baseDir, "share", "kata-initrd.zst")
		config.Hypervisor["qemu"]["kernel"] = filepath.Join(baseDir, "share", "kata-kernel")
		// Replace the kernel params entirely (and don't append) since that's
		// also what we do when calculating the launch measurement.
		config.Hypervisor["qemu"]["kernel_params"] = qemuExtraKernelParams

		config.Hypervisor["qemu"]["snp_id_block"] = snpIDBlock.IDBlock
		config.Hypervisor["qemu"]["snp_id_auth"] = snpIDBlock.IDAuth

		if debug {
			config.Hypervisor["qemu"]["enable_debug"] = true
		}
		// GPU-specific settings
		if platform == platforms.K3sQEMUSNPGPU || platform == platforms.MetalQEMUSNPGPU {
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

// ContainerdBaseConfig returns the base containerd configuration.
func ContainerdBaseConfig() config.ContainerdConfig {
	var config config.ContainerdConfig
	if err := toml.Unmarshal([]byte(containerdBaseConfig), &config); err != nil {
		panic(err) // should never happen
	}
	return config
}

// ContainerdRuntimeConfigFragment returns the containerd runtime configuration fragment.
func ContainerdRuntimeConfigFragment(baseDir, snapshotter string, platform platforms.Platform) (*config.Runtime, error) {
	cfg := config.Runtime{
		Type:                         "io.containerd.contrast-cc.v2",
		Path:                         filepath.Join(baseDir, "bin", "containerd-shim-contrast-cc-v2"),
		PodAnnotations:               []string{"io.katacontainers.*"},
		PrivilegedWithoutHostDevices: true,
	}

	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-clh-snp.toml"),
		}
		cfg.Snapshotter = snapshotter
	case platforms.MetalQEMUTDX, platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-qemu-tdx.toml"),
		}
	case platforms.MetalQEMUSNP, platforms.K3sQEMUSNP, platforms.K3sQEMUSNPGPU,
		platforms.MetalQEMUSNPGPU:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-qemu-snp.toml"),
		}
		// For GPU support, we need to pass through the CDI annotations.
		if platform == platforms.K3sQEMUSNPGPU || platform == platforms.MetalQEMUSNPGPU {
			cfg.PodAnnotations = append(cfg.PodAnnotations, "cdi.k8s.io/*")
		}
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	return &cfg, nil
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
