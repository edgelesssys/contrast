// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !runtimers

package main

import (
	"fmt"
	"path/filepath"
)

const (
	testdataSubdir = "runtime-go"
	configSuffix   = "go"
)

func upstreamFile(tarball, platform string) string {
	return filepath.Join(tarball, "opt", "kata", "share", "defaults", "kata-containers", fmt.Sprintf("configuration-%s.toml", platform))
}
