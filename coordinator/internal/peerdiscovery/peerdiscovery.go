// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build enterprise

package peerdiscovery

import (
	"log/slog"

	enterprise "github.com/edgelesssys/contrast/enterprise/coordinator/peerdiscovery"
)

// New creates a new peer discovery store.
func New(log *slog.Logger) (*enterprise.PeerStore, error) {
	return enterprise.New(log)
}
