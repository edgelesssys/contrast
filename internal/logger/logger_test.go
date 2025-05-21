// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogOutput(t *testing.T) {
	testCases := map[string]struct {
		logLevel     string
		logFormat    string
		wantMessages int
	}{
		"default": {
			wantMessages: 4,
		},
		"text warning": {
			logLevel:     "warn",
			logFormat:    "text",
			wantMessages: 3,
		},
		"json debug": {
			logLevel:     "debug",
			logFormat:    "json",
			wantMessages: 5,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			getEnv := newTestGetEnv(map[string]string{
				LogLevel:  tc.logLevel,
				LogFormat: tc.logFormat,
			})

			buf := bytes.Buffer{}

			level, err := getLogLevel(getEnv)
			require.NoError(t, err)
			handler := logHandler(getEnv)(&buf, &slog.HandlerOptions{
				Level: level,
			})
			logger := slog.New(handler)

			logger.Debug("debug")
			logger.Info("info")
			logger.Warn("warn")
			logger.Error("error")

			got := buf.String()
			lines := strings.Split(got, "\n")
			assert.Len(lines, tc.wantMessages)
			for _, line := range lines {
				if line == "" {
					continue
				}
				if tc.logFormat == "json" {
					assert.NoError(json.Unmarshal([]byte(line), &map[string]any{}))
				} else {
					assert.Error(json.Unmarshal([]byte(line), &map[string]any{}))
				}
			}
		})
	}
}

func TestGetLogLevel(t *testing.T) {
	testCases := map[string]struct {
		logLevel string
		want     slog.Level
		wantErr  bool
	}{
		"empty": {
			logLevel: "",
			want:     slog.LevelInfo,
		},
		"debug": {
			logLevel: "debug",
			want:     slog.LevelDebug,
		},
		"info with casing": {
			logLevel: "InFo",
			want:     slog.LevelInfo,
		},
		"warn": {
			logLevel: "warn",
			want:     slog.LevelWarn,
		},
		"error": {
			logLevel: "error",
			want:     slog.LevelError,
		},
		"invalid": {
			logLevel: "invalid",
			wantErr:  true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			getEnv := newTestGetEnv(map[string]string{
				LogLevel: tc.logLevel,
			})

			got, err := getLogLevel(getEnv)
			if tc.wantErr {
				assert.Error(err)
				return
			}
			require.NoError(err)
			assert.Equal(tc.want, got)
		})
	}
}

func TestLogHandler(t *testing.T) {
	testCases := map[string]struct {
		logFormat      string
		correctHandler func(slog.Handler) bool
	}{
		"empty": {
			logFormat:      "",
			correctHandler: isTextHandler,
		},
		"fallback": {
			logFormat:      "foo",
			correctHandler: isTextHandler,
		},
		"text": {
			logFormat:      "text",
			correctHandler: isTextHandler,
		},
		"json": {
			logFormat:      "jSoN",
			correctHandler: isJSONHandler,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			getEnv := newTestGetEnv(map[string]string{
				LogFormat: tc.logFormat,
			})

			got := logHandler(getEnv)(nil, nil)
			assert.True(tc.correctHandler(got))
		})
	}
}

func newTestGetEnv(environ map[string]string) func(string) string {
	return func(key string) string {
		return environ[key]
	}
}

func isTextHandler(h slog.Handler) bool {
	_, ok := h.(*slog.TextHandler)
	return ok
}

func isJSONHandler(h slog.Handler) bool {
	_, ok := h.(*slog.JSONHandler)
	return ok
}
