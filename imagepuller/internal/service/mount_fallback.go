// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !linux

package service

import (
	"log/slog"
)

func (s *ImagePullerService) createAndMountContainer(*slog.Logger, string, string) (string, error) {
	panic("GOOS does not support mounting containers")
}
