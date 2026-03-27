// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kataconfig

import (
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/pelletier/go-toml/v2"
)

var (
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

	// RuntimeNamePlaceholder is the placeholder for the per-runtime path (i.e. /opt/edgeless/contrast-cc...) in the target file paths.
	RuntimeNamePlaceholder = "@@runtimeName@@"
)

// KataRuntimeConfig returns the Kata runtime configuration.
func KataRuntimeConfig(
	baseDir string,
	platform platforms.Platform,
	qemuExtraKernelParams string,
	imagepullerConfigPath string,
	debug bool,
) (*Config, error) {
	var customContrastAnnotations []string
	var config Config
	switch {
	case platforms.IsTDX(platform):
		if err := toml.Unmarshal([]byte(kataBareMetalQEMUTDXBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}
		config.Hypervisor["qemu"]["firmware"] = filepath.Join(baseDir, "tdx", "share", "OVMF.fd")
		// We set up dm_verity in the system NixOS config.
		// Doing so again here prevents VM boots.
		config.Hypervisor["qemu"]["kernel_verity_params"] = ""
	case platforms.IsSNP(platform):
		if err := toml.Unmarshal([]byte(kataBareMetalQEMUSNPBaseConfig), &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kata runtime configuration: %w", err)
		}

		for _, productLine := range []string{"_Milan", "_Genoa"} {
			for _, annotationType := range []string{"snp_id_block", "snp_id_auth", "snp_guest_policy"} {
				customContrastAnnotations = append(customContrastAnnotations, annotationType+productLine)
			}
		}

		config.Hypervisor["qemu"]["firmware"] = filepath.Join(baseDir, "snp", "share", "OVMF.fd")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
	if debug {
		config.Agent["kata"]["enable_debug"] = true
		config.Agent["kata"]["debug_console_enabled"] = true
		config.Runtime["enable_debug"] = true
	}
	// For larger images, we've been running into timeouts in e2e tests.
	config.Agent["kata"]["dial_timeout"] = 120
	config.Runtime["create_container_timeout"] = 120
	// GPU-specific settings
	if platforms.IsGPU(platform) {
		config.Hypervisor["qemu"]["cold_plug_vfio"] = "root-port"
		// GPU images tend to be larger, so give a better default timeout that
		// allows for pulling those.
		config.Agent["kata"]["dial_timeout"] = 600
		config.Runtime["create_container_timeout"] = 600
		config.Runtime["pod_resource_api_sock"] = "/var/lib/kubelet/pod-resources/kubelet.sock"
	}

	// Use the resources installed by Contrast node-installer.
	config.Hypervisor["qemu"]["initrd"] = filepath.Join(baseDir, "share", "kata-initrd.zst")
	config.Hypervisor["qemu"]["kernel"] = filepath.Join(baseDir, "share", "kata-kernel")
	config.Hypervisor["qemu"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
	config.Hypervisor["qemu"]["rootfs_type"] = "erofs"
	config.Hypervisor["qemu"]["path"] = filepath.Join(baseDir, "bin", "qemu-system-x86_64")
	config.Hypervisor["qemu"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "bin", "qemu-system-x86_64")}
	config.Hypervisor["qemu"]["contrast_imagepuller_config"] = imagepullerConfigPath
	// TODO(katexochen): Remove after https://github.com/kata-containers/kata-containers/pull/12472 is merged.
	config.Hypervisor["qemu"]["disable_image_nvdimm"] = true

	// Force container image gust pull so we don't have to use nydus-snapshotter.
	config.Runtime["experimental_force_guest_pull"] = true
	// Replace the kernel params entirely (and don't append) since that's
	// also what we do when calculating the launch measurement.
	config.Hypervisor["qemu"]["kernel_params"] = qemuExtraKernelParams
	// Conditionally enable debug mode.
	config.Hypervisor["qemu"]["enable_debug"] = debug
	// Disable all annotations, as we don't support these. Some will mess up measurements,
	// others bypass things you can archive via correct resource declaration anyway.
	config.Hypervisor["qemu"]["enable_annotations"] = append(customContrastAnnotations, "cc_init_data")
	// Fix and align guest memory calculation.
	config.Hypervisor["qemu"]["default_memory"] = platforms.DefaultMemoryInMebiBytes(platform)
	config.Runtime["sandbox_cgroup_only"] = true
	// Currently not using the upstream encrypted emptyDir feature.
	config.Runtime["emptydir_mode"] = "shared-fs"
	// TODO: Check again why we need this and how we can avoid it.
	config.Hypervisor["qemu"]["block_device_aio"] = "threads"

	config = extraRuntimeConfig(config)

	return &config, nil
}

// Config is the configuration for the Kata runtime.
// Source: https://github.com/kata-containers/kata-containers/blob/4029d154ba0c26fcf4a8f9371275f802e3ef522c/src/runtime/pkg/katautils/Config.go
// This is a simplified version of the actual configuration.
type Config struct {
	Hypervisor map[string]hypervisorConfig `toml:"hypervisor"`
	Agent      map[string]agentConfig      `toml:"agent"`
	Image      imageConfig                 `toml:"image"`
	Factory    factoryConfig               `toml:"factory"`
	Runtime    runtimeConfig               `toml:"runtime"`
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
