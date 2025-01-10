package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"syscall"
	"unsafe"
)

func ListenAndServe(ctx context.Context, addr string, cfg *tls.Config) error {
	listenConfig := &net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var err error
			c.Control(func(fd uintptr) {
				err = syscall.SetsockoptInt(int(fd), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1)
			})
			return err
		},
	}

	l, err := listenConfig.Listen(ctx, "tcp4", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}
	defer l.Close()

	for {
		netConn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("accepting on %s: %w", l.Addr(), err)
		}
		conn, ok := netConn.(*net.TCPConn)
		if !ok {
			log.Fatalf("Listener returned unexpected connection type: %T", netConn)
		}
		upstream, err := originalDst(conn)
		// TODO: ignore own address properly
		if upstream.Port == 15006 || upstream.Port == 15007 {
			log.Printf("Cowardly refusing to forward to my own address (%s <-> %s).", conn.RemoteAddr(), upstream)
			conn.Close()
			continue
		}
		if err != nil {
			log.Printf("Ingress handling error for peer %s: %v", conn.RemoteAddr(), err)
			conn.Close()
			continue
		}
		go handle(tls.Server(conn, cfg), upstream)
	}
}

func handle(downstreamConn shutdownConn, upstream *net.TCPAddr) {
	defer downstreamConn.Close()
	log.Printf("Connecting %v <-> %v", downstreamConn.RemoteAddr(), upstream)

	upstreamConn, err := net.DialTCP("tcp4", nil, upstream)
	if err != nil {
		log.Printf("Error connecting %s <-> %s: %v", downstreamConn.RemoteAddr(), upstream, err)
		return
	}
	defer upstreamConn.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		splice(downstreamConn, upstreamConn)
		wg.Done()
	}()

	go func() {
		splice(upstreamConn, downstreamConn)
		wg.Done()
	}()

	wg.Wait()
	log.Printf("Done %v <-> %v", downstreamConn.RemoteAddr(), upstream)
}

type shutdownConn interface {
	net.Conn
	CloseWrite() error
}

func splice(to, from shutdownConn) {
	if _, err := io.Copy(to, from); err != nil {
		log.Printf("error copying data: %v", err)
		to.Close()
		from.Close()
	}
	_ = to.CloseWrite()
}

func originalDst(conn *net.TCPConn) (*net.TCPAddr, error) {
	syscallConn, err := conn.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("getting SyscallConn: %w", err)
	}
	var addr *net.TCPAddr
	controlErr := syscallConn.Control(func(fd uintptr) {
		sa := new(syscall.RawSockaddrInet4)
		var optLen uint32 = syscall.SizeofSockaddrInet4
		_, _, errno := syscall.Syscall6(
			syscall.SYS_GETSOCKOPT,
			fd,
			0,  // SOL_IP
			80, // SO_ORIGINAL_DST
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
		return nil, fmt.Errorf("getting original destination: %w", err)
	}
	return addr, err
}

func ntohs(n uint16) int {
	return int((n&0xFF)<<8 | n>>8)
}
