// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !linux

package cryptsetup

import "context"

// IsExt4 is not implemented for non-Linux systems.
func (d *Device) IsExt4(context.Context) (bool, error) {
	panic("GOOS does not support cryptesetup")
}

// MkfsExt4 is not implemented for non-Linux systems.
func (d *Device) MkfsExt4(context.Context) error {
	panic("GOOS does not support cryptesetup")
}
