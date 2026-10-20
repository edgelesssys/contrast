// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !runtimers

package upstream

import _ "embed"

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
