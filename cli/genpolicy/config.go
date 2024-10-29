// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package genpolicy

import (
	_ "embed"
	"fmt"

	"github.com/edgelesssys/contrast/internal/platforms"
)

var (
	//go:embed assets/genpolicy-microsoft
	aksGenpolicyBin []byte
	//go:embed assets/genpolicy-kata
	kataGenpolicyBin []byte
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
	// Bin is the genpolicy binary.
	Bin []byte
}

// NewConfig selects the appropriate genpolicy configuration for the target platform.
func NewConfig(platform platforms.Platform) (*Config, error) {
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		return &Config{
			Rules:    aksRules,
			Settings: aksSettings,
			Bin:      aksGenpolicyBin,
		}, nil
	case platforms.K3sQEMUSNP, platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
		return &Config{
			Rules:    kataRules,
			Settings: kataSettings,
			Bin:      kataGenpolicyBin,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported platform %s", platform)
	}
}
