// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newCLILogger(cmd *cobra.Command) (*slog.Logger, error) {
	rawLogLevel, err := cmd.Flags().GetString("log-level")
	if err != nil {
		rawLogLevel = "warn"
	}
	var level slog.Level
	switch strings.ToLower(rawLogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "":
		fallthrough
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		numericLevel, err := strconv.Atoi(rawLogLevel)
		if err != nil {
			return nil, fmt.Errorf("invalid log level: %q", rawLogLevel)
		}
		level = slog.Level(numericLevel)
	}
	opts := &slog.HandlerOptions{
		Level: level,
	}
	return slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), opts)), nil
}
