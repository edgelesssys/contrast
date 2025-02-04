// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package constants

import "github.com/google/go-sev-guest/abi"

// Version value is injected at build time.
var (
	Version                   = "0.0.0-dev"
	MicrosoftGenpolicyVersion = "0.0.0-dev"
	KataGenpolicyVersion      = "0.0.0-dev"
)

const (
	// SecretSeedSize is the size of the secret seed generated in the coordinator.
	SecretSeedSize = 64
	// SecretSeedSaltSize is the size of the secret seed salt generated in the coordinator.
	SecretSeedSaltSize = 32
	// SNPCertChainExtrasCRLKey is the UUID of the cert chain extra that contains the CRL.
	SNPCertChainExtrasCRLKey = "00569ee4-e480-4fba-bbf4-45b629901180"
)

// SNPPolicy is the default policy for the SEV-SNP platform.
var SNPPolicy = abi.SnpPolicy{
	SMT:   true,
	Debug: false,
}
