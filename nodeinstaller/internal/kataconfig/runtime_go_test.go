// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !runtimers

package kataconfig_test

import _ "embed"

var (
	//go:embed testdata/runtime-go/expected-configuration-qemu-snp.toml
	expectedConfMetalQEMUSNP []byte
	//go:embed testdata/runtime-go/expected-configuration-qemu-tdx.toml
	expectedConfMetalQEMUTDX []byte
	//go:embed testdata/runtime-go/expected-configuration-qemu-snp-gpu.toml
	expectedConfMetalQEMUSNPGPU []byte
	//go:embed testdata/runtime-go/expected-configuration-qemu-tdx-gpu.toml
	expectedConfMetalQEMUTDXGPU []byte
)
