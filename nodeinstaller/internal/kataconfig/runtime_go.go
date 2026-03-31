// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !runtimers

package kataconfig

import (
	_ "embed"

	"github.com/edgelesssys/contrast/internal/platforms"
)

var (
	// kataBareMetalQEMUTDXBaseConfig is the configuration file for the Kata runtime on bare-metal TDX
	// with QEMU.
	//
	//go:embed configuration-qemu-tdx-go.toml
	kataBareMetalQEMUTDXBaseConfig string
	// kataBareMetalQEMUSNPBaseConfig is the configuration file for the Kata runtime on bare-metal SNP
	// with QEMU.
	//
	//go:embed configuration-qemu-snp-go.toml
	kataBareMetalQEMUSNPBaseConfig string
)

func extraRuntimeConfig(config Config, platform platforms.Platform) Config {
	// Currently not using the upstream encrypted emptyDir feature.
	config.Runtime["emptydir_mode"] = "shared-fs"
	// For larger images, we've been running into timeouts in e2e tests.
	config.Runtime["create_container_timeout"] = 120
	// Force container image gust pull so we don't have to use nydus-snapshotter.
	config.Runtime["experimental_force_guest_pull"] = true

	config.Agent["kata"]["dial_timeout"] = 120

	if platforms.IsGPU(platform) {
		// GPU images tend to be larger, so give a better default timeout that
		// allows for pulling those.
		config.Agent["kata"]["dial_timeout"] = 600
		config.Runtime["create_container_timeout"] = 600
		config.Hypervisor["qemu"]["cold_plug_vfio"] = "root-port"
		config.Runtime["pod_resource_api_sock"] = "/var/lib/kubelet/pod-resources/kubelet.sock"
	}

	return config
}
