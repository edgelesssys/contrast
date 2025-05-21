// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package config

import "github.com/pelletier/go-toml/v2"

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

// Marshal encodes the configuration as TOML.
func (k *KataRuntimeConfig) Marshal() ([]byte, error) {
	return toml.Marshal(k)
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
