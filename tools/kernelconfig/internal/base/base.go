// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package base

import (
	_ "embed"
)

var (
	// BaseConfig is the base kernel config copied from kata-containers.
	//go:embed config
	BaseConfig []byte
	// BaseConfigGPU is the base kernel config with GPU enablement copied from kata-containers.
	//go:embed config-nvidia-gpu
	BaseConfigGPU []byte
)
