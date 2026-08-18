// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apitypes

import (
	"go/build"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNoExternalImports asserts that apitypes depends on the standard library only.
func TestNoExternalImports(t *testing.T) {
	// Mode 0 only scans the directory for import statements, it does not resolve them.
	pkg, err := build.ImportDir(".", 0)
	require.NoError(t, err)

	for _, imp := range pkg.Imports {
		// Standard library import paths have no dot in their first segment, because that segment cannot be a domain name.
		first, _, _ := strings.Cut(imp, "/")
		require.NotContains(t, first, ".", "apitypes must not import %q", imp)
	}
}
