// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !linux

package fileop

import (
	"os"
)

// copyAt copies the contents of src to dst.
func (o *OS) copyAt(src, dst *os.File) error {
	return o.copyTraditional(src, dst)
}
