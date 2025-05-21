// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !linux

package embedbin

import "github.com/spf13/afero"

// New returns a new installer.
func New() *RegularInstaller {
	return &RegularInstaller{fs: afero.NewOsFs()}
}
