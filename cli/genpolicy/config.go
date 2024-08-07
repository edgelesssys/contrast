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
	//go:embed assets/genpolicy-settings.json
	defaultGenpolicySettings []byte
	//go:embed assets/genpolicy-rules.rego
	aksCloudHypervisorSNPRules []byte
	//go:embed assets/allow-all.rego
	permissiveRules []byte
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
	cfg := &Config{
		Settings: defaultGenpolicySettings,
	}
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		cfg.Rules = aksCloudHypervisorSNPRules
	default:
		// TODO(burgerdev): use real rules for supported platforms.
		cfg.Rules = permissiveRules
	}
	return cfg
}
