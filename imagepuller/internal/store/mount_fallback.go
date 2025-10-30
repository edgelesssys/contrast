// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build !linux

package store

func (s *Store) Mount(where string, layerDigests ...string) error {
	panic("GOOS does not support mounting containers")
}
