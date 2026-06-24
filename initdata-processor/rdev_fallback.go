// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !linux

package main

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// devMajor returns the major device number from stat.Rdev.
func devMajor(stat *syscall.Stat_t) uint32 {
	// Off Linux, Stat_t.Rdev is not uint64 (e.g. int32 on darwin), so convert
	// it. These non-Linux variants exist only so the package builds and lints
	// on macOS; the initdata-processor itself only runs on Linux.
	return unix.Major(uint64(stat.Rdev))
}
