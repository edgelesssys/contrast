// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apitypes

import (
	"errors"
	"go/build"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// selfImportPath is the import path prefix shared by apitypes and its subpackages.
const selfImportPath = "github.com/edgelesssys/contrast/apitypes"

// TestNoExternalImports asserts that apitypes and its subpackages depend on the standard library and each other only.
func TestNoExternalImports(t *testing.T) {
	require.NoError(t, filepath.WalkDir(".", func(dir string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		// Mode 0 only scans the directory for import statements, it does not resolve them.
		pkg, err := build.ImportDir(dir, 0)
		var noGo *build.NoGoError
		if errors.As(err, &noGo) {
			return nil
		}
		require.NoError(t, err)

		for _, imp := range pkg.Imports {
			if imp == selfImportPath || strings.HasPrefix(imp, selfImportPath+"/") {
				continue
			}
			// Standard library import paths have no dot in their first segment, because that segment cannot be a domain name.
			first, _, _ := strings.Cut(imp, "/")
			require.NotContains(t, first, ".", "package %q must not import %q", dir, imp)
		}
		return nil
	}))
}
