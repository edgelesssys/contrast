// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package acpi

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteBlob writes data at its validated fw_cfg path below outputDir.
func WriteBlob(outputDir, name string, data []byte) error {
	if !fs.ValidPath(name) {
		return fmt.Errorf("invalid fw_cfg blob name %q", name)
	}
	destination := filepath.Join(outputDir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("creating directory for %q: %w", name, err)
	}
	if err := os.WriteFile(destination, data, 0o644); err != nil {
		return fmt.Errorf("writing ACPI blob %q: %w", name, err)
	}
	return nil
}
