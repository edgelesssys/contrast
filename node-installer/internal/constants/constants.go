// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package constants

import (
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/edgelesssys/contrast/node-installer/internal/config"
	"github.com/edgelesssys/contrast/node-installer/platforms"
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

	// containerdBaseConfig is the base configuration file for containerd
	//
	//go:embed containerd-config.toml
	containerdBaseConfig string
)

// CRIFQDN is the fully qualified domain name of the CRI service.
const CRIFQDN = "io.containerd.grpc.v1.cri"

// KataRuntimeConfig returns the Kata runtime configuration.
func KataRuntimeConfig(baseDir string, platform platforms.Platform, debug bool) (*config.KataRuntimeConfig, error) {
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
		return &config, nil
	case platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
		if err := toml.Unmarshal([]byte(kataBareMetalQEMUTDXBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}
		config.Hypervisor["qemu"]["path"] = filepath.Join(baseDir, "bin", "qemu-system-x86_64")
		config.Hypervisor["qemu"]["firmware"] = filepath.Join(baseDir, "shae", "OVMF_CODE.fd")
		config.Hypervisor["qemu"]["firmware_volume"] = filepath.Join(baseDir, "share", "OVMF_VARS.fd")
		config.Hypervisor["qemu"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
		config.Hypervisor["qemu"]["kernel"] = filepath.Join(baseDir, "share", "kata-kernel")
		config.Hypervisor["qemu"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "bin", "qemu-system-x86_64")}
		if debug {
			config.Hypervisor["qemu"]["enable_debug"] = true
			config.Hypervisor["qemu"]["kernel_params"] = " agent.log=debug initcall_debug"
			config.Agent["kata"]["enable_debug"] = true
			config.Agent["kata"]["debug_console_enabled"] = true
			config.Runtime["enable_debug"] = true
		}
		return &config, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
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
func ContainerdRuntimeConfigFragment(baseDir string, platform platforms.Platform) (*config.Runtime, error) {
	cfg := config.Runtime{
		Type:                         "io.containerd.contrast-cc.v2",
		Path:                         filepath.Join(baseDir, "bin", "containerd-shim-contrast-cc-v2"),
		PodAnnotations:               []string{"io.katacontainers.*"},
		PrivilegedWithoutHostDevices: true,
	}

	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		cfg.Snapshotter = "tardev"
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-clh-snp.toml"),
		}
	case platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-qemu-tdx.toml"),
		}
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	return &cfg, nil
}

// TardevSnapshotterConfigFragment returns the tardev snapshotter configuration fragment.
func TardevSnapshotterConfigFragment() config.ProxyPlugin {
	return config.ProxyPlugin{
		Type:    "snapshot",
		Address: "/run/containerd/tardev-snapshotter.sock",
	}
}
