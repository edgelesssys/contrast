// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !enterprise

package history

import (
	"fmt"

	"github.com/spf13/afero"
)

const (
	histPath = "/mnt/state/history"
)

// NewStore creates a new AferoStore backed by the default filesystem store.
func NewStore() (*AferoStore, error) {
	osFS := afero.NewOsFs()
	if err := osFS.MkdirAll(histPath, 0o755); err != nil {
		return nil, fmt.Errorf("creating history directory: %w", err)
	}
	return NewAferoStore(&afero.Afero{Fs: afero.NewBasePathFs(osFS, histPath)}), nil
}
