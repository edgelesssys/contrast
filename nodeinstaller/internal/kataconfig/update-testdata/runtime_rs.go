// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build runtimers

package main

import (
	_ "embed"
	"fmt"
	"path/filepath"
)

const (
	testdataSubdir = "runtime-rs"
	configSuffix   = "rs"
)

func upstreamFile(tarball, platform string) string {
	return filepath.Join(tarball, "opt", "kata", "share", "defaults", "kata-containers", "runtime-rs", fmt.Sprintf("configuration-%s-runtime-rs.toml", platform))
}
