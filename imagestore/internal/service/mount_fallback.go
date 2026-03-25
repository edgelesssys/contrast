// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !linux

package service

import (
	"context"
	"log/slog"

	"github.com/edgelesssys/contrast/internal/katacomponents"
)

func setupLuksAndMount(context.Context, *slog.Logger, *katacomponents.SecureMountRequest, *SecureImageStoreParams) error {
	panic("GOOS does not support mounting LUKS devices")
}
