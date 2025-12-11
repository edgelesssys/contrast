// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package securemountapi

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-ttrpc_out=. --go-ttrpc_opt=paths=source_relative imagestore.proto

// Socket is the unix socket of the imagestore API.
const Socket = "/run/guest-services/securemount.socket"
