// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !runtimers

package kataconfig

func extraRuntimeConfig(config Config) Config {
	// No extra configuration for the Go runtime.
	return config
}
