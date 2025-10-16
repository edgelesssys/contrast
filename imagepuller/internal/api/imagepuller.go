// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package api

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-ttrpc_out=. --go-ttrpc_opt=paths=source_relative imagepuller.proto

// Socket is the unix socket of the imagepuller API.
const Socket = "/run/guest-services/imagepull.socket"

// StorePathMemory is the default dir used for the store cache.
const StorePathMemory = "/run/kata-containers/image-memory"

// StorePathStorage is the preferred dir for the store cache.
// The kata-agent uses this dir as the mount point for securely mounted storage.
// It is only used as store cache when a storage device is actually available.
const StorePathStorage = "/run/kata-containers/image"

// InsecureConfigPath specifies the location at which the
// imagepuller's authentication configuration is expected.
const InsecureConfigPath = "/run/insecure-cfg/imagepuller.toml"
