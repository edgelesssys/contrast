// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package logger provides a slog.Logger that can be configured via environment variables.
// CONTRAST_LOG_LEVEL can be used to set the log level.
// CONTRAST_LOG_FORMAT can be used to set the log format.
// It also offer a slog.Handler that can be used to enable logging on a per-subsystem basis.
// CONTRAST_LOG_SUBSYSTEMS can be used to enable logging for specific subsystems.
// If CONTRAST_LOG_SUBSYSTEMS has the special value "*", all subsystems are enabled.
// Otherwise, a comma-separated list of subsystem names can be specified.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	glogger "github.com/google/logger"
)

const (
	// LogLevel is the environment variable used to set the log level.
	LogLevel = "CONTRAST_LOG_LEVEL"
	// LogFormat is the environment variable used to set the log format.
	LogFormat = "CONTRAST_LOG_FORMAT"
	// LogSubsystems is the environment variable used to enable logging for specific subsystems.
	LogSubsystems = "CONTRAST_LOG_SUBSYSTEMS"
)

// Default returns a logger configured via environment variables.
func Default() (*slog.Logger, error) {
	logLevel, err := getLogLevel(os.Getenv)
	if err != nil {
		return nil, err
	}
	logger := slog.New(logHandler(os.Getenv)(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	logger.Info("Logger initialized", "level", logLevel.String())
	if strings.Contains(os.Getenv(LogSubsystems), "google") {
		// Used by go-sev-guest/go-tdx-guest.
		glogger.SetLevel(10)
		logger.Info("Google logger initialized")
	}
	return logger, nil
}

func getLogLevel(getEnv func(string) string) (slog.Level, error) {
	logLevel := getEnv(LogLevel)
	switch strings.ToLower(logLevel) {
	case "debug":
		return slog.LevelDebug, nil
	case "", "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	}

	return slog.Level(0), fmt.Errorf("invalid log level: %q", logLevel)
}

func logHandler(getEnv func(string) string) func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	switch strings.ToLower(getEnv(LogFormat)) {
	case "json":
		return func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
			return slog.NewJSONHandler(w, opts)
		}
	}
	return func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
		return slog.NewTextHandler(w, opts)
	}
}
