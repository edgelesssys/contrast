package execserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/mdlayher/vsock"
)

// Handler can handle a single command execution.
type Handler interface {
	// Handle receives a command over the net.Conn and executes it until it terminates or the context expires.
	//
	// The net.Conn is closed after this function returns.
	Handle(context.Context, net.Conn)
}

// Server receives connections from a listener and runs the embedded Handler for every connection.
type Server struct {
	log *slog.Logger
	h   Handler
}

// NewServer constructs a new server that dispatches to the given Handler.
func NewServer(log *slog.Logger, h Handler) *Server {
	return &Server{log: log, h: h}
}

// ListenAndServeVsock creates a VSOCK listener and serves connections until the context expires.
func (s *Server) ListenAndServeVsock(ctx context.Context, port uint32) error {
	l, err := vsock.Listen(port, nil)
	if err != nil {
		return fmt.Errorf("listening on vsock port %d: %w", port, err)
	}
	defer l.Close()
	return s.Serve(ctx, l)
}

// Serve serves connections from the listener until the context expires.
func (s *Server) Serve(ctx context.Context, l net.Listener) error {
	conns := make(chan net.Conn)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				s.log.Error("accept failed", "error", err)
				return
			}
			conns <- conn
		}
	}()

	for {
		select {
		case conn := <-conns:
			s.log.Info("Accepted connection", "peer", conn.RemoteAddr())
			go s.h.Handle(ctx, conn)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
