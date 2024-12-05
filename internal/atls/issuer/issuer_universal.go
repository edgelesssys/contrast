// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !linux

package issuer

import (
	"log/slog"

	"github.com/edgelesssys/contrast/internal/atls"
)

// New creates an attestation issuer for the current platform.
func New(_ *slog.Logger) (atls.Issuer, error) {
	panic("issuing is only supported on Linux")
}
