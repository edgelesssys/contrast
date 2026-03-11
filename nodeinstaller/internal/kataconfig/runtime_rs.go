// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build runtimers

package kataconfig

func extraRuntimeConfig(config Config) Config {
	// runtime-rs specific configuration.
	config.Runtime["name"] = "virt_container"
	config.Runtime["hypervisor_name"] = "qemu"
	config.Runtime["agent_name"] = "kata"
	return config
}
