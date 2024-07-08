// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package snp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrustedRoots(t *testing.T) {
	roots := trustedRoots()
	assert.Contains(t, roots, "Milan")
	assert.Contains(t, roots, "Genoa")
}
