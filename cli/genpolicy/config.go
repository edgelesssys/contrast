// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package genpolicy

import (
	_ "embed"

	"github.com/edgelesssys/contrast/internal/platforms"
)

var (
	//go:embed assets/genpolicy
	genpolicyBin []byte
	//go:embed assets/genpolicy-settings-microsoft.json
	aksSettings []byte
	//go:embed assets/genpolicy-settings-kata.json
	kataSettings []byte
	//go:embed assets/genpolicy-rules-microsoft.rego
	aksRules []byte
	//go:embed assets/genpolicy-rules-kata.rego
	kataRules []byte
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
			Rules:    aksRules,
			Settings: aksSettings,
		}
	case platforms.K3sQEMUSNP, platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
		return &Config{
			Rules:    kataRules,
			Settings: kataSettings,
		}
	default:
		return nil
	}
}
