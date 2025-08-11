// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package api

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-ttrpc_out=. --go-ttrpc_opt=paths=source_relative securemount.proto

// Socket is the unix socket of the securemount API.
const Socket = "/run/confidential-containers/securemount.sock"
