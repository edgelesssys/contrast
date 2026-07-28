package execserver

import (
	"bytes"
	"io"
	"log/slog"
	"net"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestRun(t *testing.T) {
	for name, tc := range map[string]struct {
		stdin string
		cmd   []string

		wantCode   int
		wantStdout string
		wantStderr string
	}{
		"echo to stdout": {
			cmd:        []string{"echo", "hello world"},
			wantStdout: "hello world\n",
		},
		"echo to stderr": {
			cmd:        []string{"sh", "-c", `echo "hello world" >&2`},
			wantStderr: "hello world\n",
		},
		"mirror stdin": {
			cmd:        []string{"cat"},
			stdin:      "ping",
			wantStdout: "ping",
		},
		"non-zero exit code": {
			cmd:      []string{"sh", "-c", "exit 5"},
			wantCode: 5,
		},
		"signaled": {
			cmd:      []string{"sh", "-c", "kill $$"},
			wantCode: int(128 + syscall.SIGTERM),
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			log := slog.Default()

			serverConn, clientConn := net.Pipe()
			serverDone := make(chan struct{})

			go func() {
				defer close(serverDone)
				NewHandler(log.With("subsystem", "handler")).Handle(t.Context(), serverConn)
			}()

			var stdoutBuf, stderrBuf bytes.Buffer
			code, err := Run(t.Context(), log.With("subsystem", "runner"), clientConn, tc.cmd, strings.NewReader(tc.stdin), &closer{&stdoutBuf}, &closer{&stderrBuf})
			require.NoError(err)
			assert.Equal(tc.wantCode, code)
			assert.Equal(tc.wantStdout, stdoutBuf.String())
			assert.Equal(tc.wantStderr, stderrBuf.String())
			<-serverDone
		})
	}
}

type closer struct {
	io.Writer
}

func (c *closer) Close() error {
	return nil
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
