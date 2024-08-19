// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package constants

import (
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/config"
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

	// RuntimeNamePlaceholder is the placeholder for the per-runtime path (i.e. /opt/edgeless/contrast-cc...) in the target file paths.
	RuntimeNamePlaceholder = "@@runtimeName@@"
)

// CRIFQDN is the fully qualified domain name of the CRI service.
const CRIFQDN = "io.containerd.grpc.v1.cri"

// KataRuntimeConfig returns the Kata runtime configuration.
func KataRuntimeConfig(baseDir string, platform platforms.Platform, qemuExtraKernelParams string, debug bool) (*config.KataRuntimeConfig, error) {
	var config config.KataRuntimeConfig
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		if err := toml.Unmarshal([]byte(kataCLHSNPBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}
		config.Hypervisor["clh"]["path"] = filepath.Join(baseDir, "bin", "cloud-hypervisor-snp")
		config.Hypervisor["clh"]["igvm"] = filepath.Join(baseDir, "share", "kata-containers-igvm.img")
		config.Hypervisor["clh"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
		config.Hypervisor["clh"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "bin", "cloud-hypervisor-snp")}
		config.Hypervisor["clh"]["enable_debug"] = debug
	case platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
		if err := toml.Unmarshal([]byte(kataBareMetalQEMUTDXBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}
		config.Hypervisor["qemu"]["path"] = filepath.Join(baseDir, "tdx", "bin", "qemu-system-x86_64")
		config.Hypervisor["qemu"]["firmware"] = filepath.Join(baseDir, "tdx", "share", "OVMF.fd")
		config.Hypervisor["qemu"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
		config.Hypervisor["qemu"]["kernel"] = filepath.Join(baseDir, "share", "kata-kernel")
		config.Hypervisor["qemu"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "tdx", "bin", "qemu-system-x86_64")}
		config.Hypervisor["qemu"]["shared_fs"] = "none"
		kernelParams := qemuExtraKernelParams
		if debug {
			config.Hypervisor["qemu"]["enable_debug"] = true
		}
		// Replace the kernel params entirely (and don't append) since that's
		// also what we do when calculating the launch measurement.
		config.Hypervisor["qemu"]["kernel_params"] = kernelParams
	case platforms.K3sQEMUSNP:
		if err := toml.Unmarshal([]byte(kataBareMetalQEMUSNPBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}
		config.Hypervisor["qemu"]["path"] = filepath.Join(baseDir, "snp", "bin", "qemu-system-x86_64")
		config.Hypervisor["qemu"]["firmware"] = filepath.Join(baseDir, "snp", "share", "OVMF.fd")
		config.Hypervisor["qemu"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
		config.Hypervisor["qemu"]["kernel"] = filepath.Join(baseDir, "share", "kata-kernel")
		delete(config.Hypervisor["qemu"], "initrd")
		config.Hypervisor["qemu"]["block_device_aio"] = "threads"
		config.Hypervisor["qemu"]["shared_fs"] = "none"
		config.Hypervisor["qemu"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "snp", "bin", "qemu-system-x86_64")}
		config.Hypervisor["qemu"]["rootfs_type"] = "erofs"
		kernelParams := qemuExtraKernelParams
		if debug {
			config.Hypervisor["qemu"]["enable_debug"] = true
		}
		// Replace the kernel params entirely (and don't append) since that's
		// also what we do when calculating the launch measurement.
		config.Hypervisor["qemu"]["kernel_params"] = kernelParams
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
		Snapshotter:                  snapshotter,
	}

	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-clh-snp.toml"),
		}
	case platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-qemu-tdx.toml"),
		}
	case platforms.K3sQEMUSNP:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-qemu-snp.toml"),
		}
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	return &cfg, nil
}
