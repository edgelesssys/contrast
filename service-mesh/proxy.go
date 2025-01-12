package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sync/errgroup"
)

type Wrapper func(shutdownConn) shutdownConn

type Director func(downstreamConn *net.TCPConn) (string, error)

type Listener struct {
	Transparent       bool
	UpstreamWrapper   Wrapper
	DownstreamWrapper Wrapper
	Director          Director
}

func (l *Listener) ListenAndServe(ctx context.Context, addr string) error {
	listenConfig := &net.ListenConfig{}
	if l.Transparent {
		listenConfig.Control = func(_, _ string, c syscall.RawConn) error {
			var err error
			controlErr := c.Control(func(fd uintptr) {
				err = syscall.SetsockoptInt(int(fd), SOL_IP, IP_TRANSPARENT, 1)
			})
			return errors.Join(err, controlErr)
		}
	}

	listener, err := listenConfig.Listen(ctx, "tcp4", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}
	defer listener.Close()

	for {
		netConn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accepting on %s: %w", listener.Addr(), err)
		}
		conn, ok := netConn.(*net.TCPConn)
		if !ok {
			panic(fmt.Sprintf("Listener returned unexpected connection type: %T", netConn))
		}

		go func() {
			if err := l.handle(ctx, conn); err != nil {
				log.Printf("Error handling downstream %s: %v", conn.RemoteAddr(), err)
			}
		}()
	}
}

func (l *Listener) handle(ctx context.Context, downstreamConn *net.TCPConn) error {
	upstream, err := l.Director(downstreamConn)
	if err != nil {
		return fmt.Errorf("getting upstream address for downstream %s: %w", downstreamConn.RemoteAddr(), err)
	}

	d := &net.Dialer{}
	upstreamConn, err := d.DialContext(ctx, "tcp4", upstream)
	if err != nil {
		downstreamConn.Close()
		return fmt.Errorf("dialing %s: %w", upstream, err)
	}

	var wrappedUpstream shutdownConn = upstreamConn.(*net.TCPConn)
	var wrappedDownstream shutdownConn = downstreamConn

	if l.UpstreamWrapper != nil {
		wrappedUpstream = l.UpstreamWrapper(wrappedUpstream)
	}

	if l.DownstreamWrapper != nil {
		wrappedDownstream = l.DownstreamWrapper(wrappedDownstream)
	}

	return splice(ctx, wrappedDownstream, wrappedUpstream)
}

type shutdownConn interface {
	io.ReadWriteCloser
	CloseWrite() error
}

func splice(ctx context.Context, a, b shutdownConn) error {
	defer a.Close()
	defer b.Close()
	eg, ctx := errgroup.WithContext(ctx)

	// TODO(burgerdev): make copy routines cancellable
	eg.Go(func() error {
		_, err := io.Copy(a, b)
		return errors.Join(err, a.CloseWrite())
	})

	eg.Go(func() error {
		_, err := io.Copy(b, a)
		return errors.Join(err, b.CloseWrite())
	})

	// TODO(burgerdev): handle idle connections?
	return eg.Wait()
}

func originalDst(conn *net.TCPConn) (string, error) {
	syscallConn, err := conn.SyscallConn()
	if err != nil {
		return "", fmt.Errorf("getting SyscallConn: %w", err)
	}
	var addr *net.TCPAddr
	controlErr := syscallConn.Control(func(fd uintptr) {
		sa := new(syscall.RawSockaddrInet4)
		var optLen uint32 = syscall.SizeofSockaddrInet4
		_, _, errno := syscall.Syscall6(
			syscall.SYS_GETSOCKOPT,
			fd,
			SOL_IP,
			SO_ORIGINAL_DST,
			uintptr(unsafe.Pointer(sa)),
			uintptr(unsafe.Pointer(&optLen)),
			0,
		)
		if errno != 0 {
			err = fmt.Errorf("getsockopt syscall error (%d): %w", int(errno), errno)
			return
		}

		addr = &net.TCPAddr{
			IP:   net.IPv4(sa.Addr[0], sa.Addr[1], sa.Addr[2], sa.Addr[3]),
			Port: ntohs(sa.Port),
		}
	})
	if controlErr != nil {
		return "", fmt.Errorf("getting original destination: %w", err)
	}
	return addr.String(), err
}

func ntohs(n uint16) int {
	return int((n&0xFF)<<8 | n>>8)
}
