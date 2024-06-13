// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// NewNamed returns a new logger with the given name, using the
// handler from the previous logger.
func NewNamed(logger *slog.Logger, name string) *slog.Logger {
	return slog.New(NewHandler(logger.Handler(), name))
}

// Handler is a slog.Handler that can be used to enable logging on a per-subsystem basis.
type Handler struct {
	inner     slog.Handler
	subsystem string
	enabled   bool
	level     slog.Level
}

// NewHandler returns a new Handler.
func NewHandler(inner slog.Handler, subsystem string) *Handler {
	handler := &Handler{
		inner:     inner.WithGroup(subsystem),
		subsystem: subsystem,
		enabled:   subsystemEnvEnabled(os.Getenv, subsystem),
		level:     slog.LevelWarn,
	}
	slog.New(handler).Info("Subsystem logger initialized", "subsystem", subsystem, "state", handler.state())
	return handler
}

// Enabled returns true if the given level is enabled.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return (h.enabled || level >= h.level) && h.inner.Enabled(ctx, level)
}

// Handle handles the given record.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	return h.inner.Handle(ctx, record)
}

// WithAttrs returns a new Handler with the given attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		inner:     h.inner.WithAttrs(attrs),
		subsystem: h.subsystem,
		enabled:   h.enabled,
	}
}

// WithGroup returns a new Handler with the given group.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		inner:     h.inner.WithGroup(name),
		subsystem: h.subsystem,
		enabled:   h.enabled,
	}
}

func (h *Handler) state() string {
	if h.enabled {
		return "enabled"
	}
	return "disabled"
}

func subsystemEnvEnabled(getEnv func(string) string, subsystem string) bool {
	return subsystemAllowListMatch(subsystem, getEnv(LogSubsystems))
}

func subsystemAllowListMatch(subsystem string, allowList string) bool {
	if allowList == "*" {
		return true
	}
	for _, allow := range strings.Split(allowList, ",") {
		allow = strings.ToLower(strings.TrimSpace(allow))
		if allow == subsystem {
			return true
		}
	}
	return false
}
