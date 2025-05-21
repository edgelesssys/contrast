// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package embedbin provides a portable way to install embedded binaries.
//
// The Install function creates a temporary file and writes the contents to it.
package embedbin

// Installed is a handle to an installed binary.
// Users must call Uninstall to clean it up.
type Installed interface {
	Path() string
	IsRegular() bool
	Uninstall() error
}
