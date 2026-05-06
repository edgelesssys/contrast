// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package logcollect provides a thin Go wrapper around the get-logs script
// for e2e tests that want to collect cluster + host logs synchronously,
// before any test cleanup tears down the namespaces being collected from.
package logcollect

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Download invokes `get-logs download` against namespaceFile, blocking until
// log collection completes or ctx is canceled.
//
// If sinceFloor is non-zero, it is passed to get-logs as a lower bound for
// the host-log --since timestamp via the LOG_COLLECT_SINCE_FLOOR env var.
func Download(ctx context.Context, namespaceFile string, sinceFloor time.Time) error {
	cmd := exec.CommandContext(ctx, "get-logs", "download", namespaceFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if !sinceFloor.IsZero() {
		cmd.Env = append(os.Environ(), "LOG_COLLECT_SINCE_FLOOR="+sinceFloor.UTC().Format(time.RFC3339))
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running get-logs download %s: %w", namespaceFile, err)
	}
	return nil
}
