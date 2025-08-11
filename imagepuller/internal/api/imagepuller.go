// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package api

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-ttrpc_out=. --go-ttrpc_opt=paths=source_relative imagepuller.proto

// Socket is the unix socket of the imagepuller API.
const Socket = "/run/confidential-containers/imagepuller.sock"
