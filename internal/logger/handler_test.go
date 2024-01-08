package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlerOutput(t *testing.T) {
	testCases := map[string]struct {
		subsystem        string
		subsystemEnvList string
		wantMessages     int
	}{
		"star": {
			subsystem:        "foo",
			subsystemEnvList: "*",
			wantMessages:     2, // message and empty line
		},
		"match": {
			subsystem:        "foo",
			subsystemEnvList: "foo,bar,baz",
			wantMessages:     2, // message and empty line
		},
		"no match": {
			subsystem:    "foo",
			wantMessages: 1, // empty line
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			getEnv := newTestGetEnv(map[string]string{
				LogSubsystems: tc.subsystemEnvList,
			})

			buf := bytes.Buffer{}

			handler := &Handler{
				inner:     slog.NewJSONHandler(&buf, nil).WithGroup(tc.subsystem),
				subsystem: tc.subsystem,
				enabled:   subsystemEnvEnabled(getEnv, tc.subsystem),
			}
			logger := slog.New(handler)

			logger.Info("info", "key", "value")

			got := buf.String()
			lines := strings.Split(got, "\n")
			assert.Len(lines, tc.wantMessages)
			for _, line := range lines {
				if line == "" {
					continue
				}
				assert.Contains(line, tc.subsystem)
			}
		})
	}
}

func TestSubsystemEnvEnabled(t *testing.T) {
	testCases := map[string]struct {
		subsystem        string
		subsystemEnvList string
		wantEnabled      bool
	}{
		"empty with star": {
			subsystem:        "",
			subsystemEnvList: "*",
			wantEnabled:      true,
		},
		"value with star": {
			subsystem:        "foo",
			subsystemEnvList: "*",
			wantEnabled:      true,
		},
		"empty list": {
			subsystem:        "bar",
			subsystemEnvList: "",
		},
		"match": {
			subsystem:        "bar",
			subsystemEnvList: "foo,bar,baz",
			wantEnabled:      true,
		},
		"no match": {
			subsystem:        "bar",
			subsystemEnvList: "foo,baz",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			getEnv := newTestGetEnv(map[string]string{
				LogSubsystems: tc.subsystemEnvList,
			})

			got := subsystemEnvEnabled(getEnv, tc.subsystem)
			assert.Equal(tc.wantEnabled, got)
		})
	}
}
