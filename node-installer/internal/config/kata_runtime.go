// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package config

// KataRuntimeConfig is the configuration for the Kata runtime.
// Source: https://github.com/kata-containers/kata-containers/blob/4029d154ba0c26fcf4a8f9371275f802e3ef522c/src/runtime/pkg/katautils/config.go
// This is a simplified version of the actual configuration.
type KataRuntimeConfig struct {
	Hypervisor map[string]Hypervisor
	Agent      map[string]Agent
	Image      Image
	Factory    Factory
	Runtime    KataRuntime
}

// Image is the configuration for the image.
type Image map[string]any

// Factory is the configuration for the factory.
type Factory map[string]any

// Hypervisor is the configuration for the hypervisor.
type Hypervisor map[string]any

// KataRuntime is the configuration for the Kata runtime.
type KataRuntime map[string]any

// Agent is the configuration for the agent.
type Agent map[string]any
