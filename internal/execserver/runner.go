package execserver

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
)

// Run runs a command at the remote end of conn.
//
// It forwards stdin to the remote command, and pipes the remote output to the local writers.
// After the command completes, Run returns the exit code.
func Run(_ context.Context, log *slog.Logger, conn net.Conn, cmd []string, stdin io.Reader, stdout, stderr io.WriteCloser) (int, error) {
	defer conn.Close()

	// Initiate the command execution by sending the argv to execute first.
	if err := WriteMessage(conn, &Request{Cmd: cmd}); err != nil {
		return 0, fmt.Errorf("sending command: %w", err)
	}

	// Forward stdin in a simple goroutine: it might be open, but not required by the remote
	// command. In that case, we would block forever on the Read although the remote command
	// completed.
	go func() {
		buf := make([]byte, readBufSize)
		for {
			n, err := stdin.Read(buf)
			// According to io.Reader, we might encounter n>0 and EOF in the same read.
			if n > 0 {
				if sendErr := WriteMessage(conn, &Request{Stdin: &StreamEvent{Data: buf[:n]}}); sendErr != nil {
					log.Error("Encountered error while forwarding stdin", "error", err)
					return
				}
			}
			if err == io.EOF {
				if sendErr := WriteMessage(conn, &Request{Stdin: &StreamEvent{EOF: true}}); sendErr != nil {
					log.Error("Encountered error while closing remote stdin", "error", err)
				}
				return
			} else if err != nil {
				if sendErr := WriteMessage(conn, &Request{Stdin: &StreamEvent{Error: err.Error()}}); sendErr != nil {
					log.Error("Encountered error while reporting stdin read error", "error", err)
				}
				return
			}
		}
	}()

	for {
		var resp Response
		if err := ReadMessage(conn, &resp); err != nil {
			return 0, fmt.Errorf("reading response: %w", err)
		}

		switch {
		case resp.Stdout != nil:
			if err := writeEvent(stdout, resp.Stdout); err != nil {
				return 0, err
			}
		case resp.Stderr != nil:
			if err := writeEvent(stderr, resp.Stderr); err != nil {
				return 0, err
			}
		case resp.ExitStatus != nil:
			return exitCode(resp.ExitStatus)
		}
	}
}

func writeEvent(dst io.WriteCloser, e *StreamEvent) error {
	if len(e.Data) > 0 {
		if _, err := dst.Write(e.Data); err != nil {
			return fmt.Errorf("writing to local stream: %w", err)
		}
	} else if e.EOF {
		return dst.Close()
	} else {
		return fmt.Errorf("remote stream error: %s", e.Error)
	}
	return nil
}

// exitCode translates a remote ExitStatus into a process exit code.
func exitCode(status *ExitStatus) (int, error) {
	var exitCode int
	switch {
	case status.Terminated != nil:
		exitCode = *status.Terminated
	case status.Signaled != nil:
		// Convention originating in bash which users are accustomed to.
		// https://www.gnu.org/software/bash/manual/html_node/Exit-Status.html
		exitCode = 128 + *status.Signaled
	default:
		return -1, fmt.Errorf("remote command ended with unknown status: %s", status.Error)
	}
	return exitCode, nil
}
