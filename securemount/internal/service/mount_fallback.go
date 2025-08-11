// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !linux

package service

import (
	"context"
	"log/slog"

	"github.com/edgelesssys/contrast/securemount/internal/api"
)

func setupLuksAndMount(context.Context, *slog.Logger, *api.SecureMountRequest, *SecureMountParams) error {
	panic("GOOS does not support mounting LUKS devices")
}
