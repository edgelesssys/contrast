// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build runtimers

package kataconfig

import (
	_ "embed"

	"github.com/edgelesssys/contrast/internal/platforms"
)

var (
	// kataBareMetalQEMUTDXBaseConfig is the configuration file for the Kata runtime on bare-metal TDX
	// with QEMU.
	//
	//go:embed configuration-qemu-tdx-rs.toml
	kataBareMetalQEMUTDXBaseConfig string
	// kataBareMetalQEMUSNPBaseConfig is the configuration file for the Kata runtime on bare-metal SNP
	// with QEMU.
	//
	//go:embed configuration-qemu-snp-rs.toml
	kataBareMetalQEMUSNPBaseConfig string
)

func extraRuntimeConfig(config Config, platform platforms.Platform) Config {
	config.Runtime["name"] = "virt_container"
	config.Runtime["hypervisor_name"] = "qemu"
	config.Runtime["agent_name"] = "kata"
	config.Runtime["experimental"] = []string{"force_guest_pull"}

	config.Agent["kata"]["dial_timeout_ms"] = 1000
	config.Agent["kata"]["reconnect_timeout_ms"] = 60000
	config.Agent["kata"]["create_container_timeout"] = 120

	return config
}
