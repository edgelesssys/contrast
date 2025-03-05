// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !enterprise

package peerdiscovery

import (
	"context"
	"log/slog"
)

// New is a stub for the enterprise version of the function.
func New(_ *slog.Logger) (*PeerStore, error) {
	return &PeerStore{}, nil
}

// PeerStore is a stub for the enterprise version of the struct.
type PeerStore struct{}

// Run is a stub for the enterprise version of the function.
func (p *PeerStore) Run(_ context.Context) error {
	return nil
}

// Stop is a stub for the enterprise version of the function.
func (p *PeerStore) Stop() {}
