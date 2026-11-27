// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package userapi

import "errors"

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative userapi.proto

// Port is the port of the coordinator API.
const Port = "1313"

// ErrNoManifest is returned when a manifest is needed but not present.
var ErrNoManifest = errors.New("no manifest configured")
