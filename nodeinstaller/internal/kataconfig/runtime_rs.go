// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build runtimers

package kataconfig

import _ "embed"

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

func extraRuntimeConfig(config Config) Config {
	// runtime-rs specific configuration.
	config.Runtime["name"] = "virt_container"
	config.Runtime["hypervisor_name"] = "qemu"
	config.Runtime["agent_name"] = "kata"
	return config
}
