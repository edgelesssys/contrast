// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package katacomponents

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-ttrpc_out=. --go-ttrpc_opt=paths=source_relative katacomponents.proto

//
// Imagepuller
//

// ImagepullSocket is the unix socket of the imagepuller API.
const ImagepullSocket = "/run/guest-services/imagepull.socket"

// StorePathMemory is the default dir used for the store cache.
const StorePathMemory = "/run/kata-containers/image-memory"

// StorePathStorage is the preferred dir for the store cache.
// The kata-agent uses this dir as the mount point for securely mounted storage.
// It is only used as store cache when a storage device is actually available.
const StorePathStorage = "/run/kata-containers/image"

// InsecureConfigPath specifies the location at which the
// imagepuller's authentication configuration is expected.
const InsecureConfigPath = "/run/insecure-cfg/imagepuller.toml"

//
// Imagestore
//

// SecuremountSocket is the unix socket of the imagestore API.
const SecuremountSocket = "/run/guest-services/securemount.socket"
