// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !linux

package store

import (
	gcr "github.com/google/go-containerregistry/pkg/v1"
)

// Mount is Linux-specific.
func (s *Store) Mount(string, ...gcr.Hash) error {
	panic("GOOS does not support mounting containers")
}
