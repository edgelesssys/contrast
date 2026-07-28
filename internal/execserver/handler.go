package execserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os/exec"
	"sync"
	"syscall"
)

type handler struct {
	log *slog.Logger
}

// NewHandler creates the default handler for executing commands for a remote peer.
func NewHandler(log *slog.Logger) Handler {
	return &handler{log: log}
}

// Handle runs a single command over the given connection.
func (h *handler) Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	// connMu guards concurrent writes to conn
	connMu := &sync.Mutex{}

	var req Request
	if err := ReadMessage(conn, &req); err != nil {
		h.log.Warn("Could not read initial message", "error", err)
		return
	}
	if len(req.Cmd) == 0 {
		slog.Warn("First request did not contain a command")
		return
	}

	cmd := exec.CommandContext(ctx, req.Cmd[0], req.Cmd[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		h.log.Warn("Could not get command's stdin", "error", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		h.log.Warn("Could not get command's stdout", "error", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		h.log.Warn("Could not get command's stderr", "error", err)
		return
	}

	if err := cmd.Start(); err != nil {
		h.log.Warn("Could not start command", "error", err)
		msg := &Response{
			ExitStatus: &ExitStatus{Error: err.Error()},
		}
		if err := h.writeResponse(conn, connMu, msg); err != nil {
			h.log.Warn("Could not send error report", "error", err)
		}
		return
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		h.connToStdin(conn, stdin)
	})
	wg.Go(func() {
		h.pipeOutput(conn, connMu, stdout, func(e *StreamEvent) *Response {
			return &Response{Stdout: e}
		})
	})
	wg.Go(func() {
		h.pipeOutput(conn, connMu, stderr, func(e *StreamEvent) *Response {
			return &Response{Stderr: e}
		})
	})

	// If the command terminates, all streams will close and the pipe functions will complete.
	// Until then, we want to keep forwarding stream data.
	wg.Wait()

	status := waitStatus(cmd.Wait())
	if err := h.writeResponse(conn, connMu, &Response{ExitStatus: status}); err != nil {
		h.log.Warn("Could not send exit status", "error", err)
	}
	h.log.Info("command finished", "status", status)
}

func (h *handler) connToStdin(conn net.Conn, stdin io.WriteCloser) {
	defer h.log.Info("stdin done")
	defer stdin.Close()
	for {
		var req Request
		if err := ReadMessage(conn, &req); err != nil {
			h.log.Warn("Could not read message", "error", err)
			return
		}
		h.log.Info("received message", "message", req)
		if req.Stdin == nil {
			h.log.Warn("Protocol violation: a subsequent message did not have Stdin set", "message", req)
			return
		} else if req.Stdin.EOF {
			return
		} else if len(req.Stdin.Data) > 0 {
			if _, err := stdin.Write(req.Stdin.Data); err != nil {
				h.log.Warn("Could not write to process stdin", "error", err)
				return
			}
		} else {
			h.log.Warn("Encountered remote error receiving stdin", "error", req.Stdin.Error)
			return
		}
	}
}

func (h *handler) pipeOutput(conn net.Conn, writeMu *sync.Mutex, r io.Reader, wrap func(*StreamEvent) *Response) {
	buf := make([]byte, readBufSize)
	keepGoing := true
	for keepGoing {
		evt := &StreamEvent{}
		n, err := r.Read(buf)
		if errors.Is(err, io.EOF) {
			evt.EOF = true
			keepGoing = false
		} else if err != nil {
			evt.Error = err.Error()
			keepGoing = false
		} else {
			evt.Data = buf[:n]
		}
		if err := h.writeResponse(conn, writeMu, wrap(evt)); err != nil {
			h.log.Warn("Could not send streaming event", "error", err)
			keepGoing = false
		}
	}
}

func (h *handler) writeResponse(conn net.Conn, writeMu *sync.Mutex, resp *Response) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	return WriteMessage(conn, resp)
}

// waitStatus translates the result of cmd.Wait into an ExitStatus.
func waitStatus(waitErr error) *ExitStatus {
	if waitErr == nil {
		code := 0
		return &ExitStatus{Terminated: &code}
	}

	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		return &ExitStatus{Error: waitErr.Error()}
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return &ExitStatus{Error: waitErr.Error()}
	}
	if status.Signaled() {
		sig := int(status.Signal())
		return &ExitStatus{Signaled: &sig}
	}
	code := status.ExitStatus()
	return &ExitStatus{Terminated: &code}
}
