// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// package resolvconf provides means to wait until /etc/resolv.conf is configured with real
// nameservers. This is working around Go's default resolver behavior which only reads the
// resolv.conf file every 5s, even if it's not valid.
//
// This package can be removed safely after upgrading to Go 1.27, or a version that contains
// https://go-review.googlesource.com/c/go/+/741140.
package resolvconf

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Wait waits until /etc/resolv.conf contains at least one nameserver entry, or the context expires.
//
// The function will continuously read /etc/resolv.conf in 500ms intervals.
func Wait(ctx context.Context, log *slog.Logger) error {
	waiter := &waiter{
		file:   "/etc/resolv.conf",
		log:    log,
		period: 500 * time.Millisecond,
	}
	return waiter.wait(ctx)
}

type waiter struct {
	file   string
	log    *slog.Logger
	period time.Duration
}

func (w *waiter) wait(ctx context.Context) error {
	ticker := time.NewTicker(w.period)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			content, err := os.ReadFile(w.file)
			if err != nil {
				w.log.Error("could not read DNS configuration", "file", w.file, "error", err)
				continue
			}
			if hasNameserver(content) {
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("context expired while waiting for nameservers in %q: %w", w.file, ctx.Err())
		}
	}
}

func hasNameserver(content []byte) bool {
	for line := range bytes.SplitSeq(content, []byte{'\n'}) {
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("nameserver")) {
			return true
		}
	}
	return false
}
