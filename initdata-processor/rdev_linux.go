// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package main

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// devMajor returns the major device number from stat.Rdev.
func devMajor(stat *syscall.Stat_t) uint32 {
	return unix.Major(stat.Rdev)
}
