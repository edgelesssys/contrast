// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package constants

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
)
