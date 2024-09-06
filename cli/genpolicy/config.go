// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package genpolicy

import (
	_ "embed"

	"github.com/edgelesssys/contrast/internal/platforms"
)

var (
	//go:embed assets/microsoft/genpolicy
	genpolicyBin []byte
	//go:embed assets/microsoft/genpolicy-settings.json
	aksGenpolicySettings []byte
	//go:embed assets/microsoft/genpolicy-rules.rego
	aksCloudHypervisorSNPRules []byte

	//go:embed assets/kata/genpolicy
	kataGenpolicyBin []byte
	//go:embed assets/kata/genpolicy-settings.json
	kataGenpolicySettings []byte
	//go:embed assets/kata/genpolicy-rules.rego
	kataCloudHypervisorSNPRules []byte
)

// Config contains configuration files for genpolicy.
type Config struct {
	// Rules is a Rego module that verifies agent requests.
	Rules []byte
	// Settings is a json config file that holds platform-specific configuration.
	Settings []byte
}

// NewConfig selects the appropriate genpolicy configuration for the target platform.
func NewConfig(platform platforms.Platform) *Config {
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		return &Config{
			Rules:    aksCloudHypervisorSNPRules,
			Settings: aksGenpolicySettings,
		}
	default:
		return &Config{
			Rules:    kataCloudHypervisorSNPRules,
			Settings: kataGenpolicySettings,
		}
	}
}
