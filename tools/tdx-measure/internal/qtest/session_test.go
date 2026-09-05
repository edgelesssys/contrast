// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package qtest

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestAcceptConnection(t *testing.T) {
	clientConnection, serverConnection := net.Pipe()
	defer clientConnection.Close()
	listener := newTestListener()
	listener.results <- testAcceptResult{connection: serverConnection}

	connection, err := acceptConnection(context.Background(), listener, make(chan error))
	if err != nil {
		t.Fatalf("acceptConnection: %v", err)
	}
	if err := connection.Close(); err != nil {
		t.Fatalf("closing accepted connection: %v", err)
	}
}

func TestAcceptConnectionReportsProcessExit(t *testing.T) {
	listener := newTestListener()
	defer listener.Close()
	want := errors.New("QEMU failed")
	processDone := make(chan error, 1)
	processDone <- want
	close(processDone)
	if _, err := acceptConnection(context.Background(), listener, processDone); !errors.Is(err, want) {
		t.Fatalf("acceptConnection error = %v, want wrapped %v", err, want)
	}
}

type testAcceptResult struct {
	connection net.Conn
	err        error
}

type testListener struct {
	results chan testAcceptResult
}

func newTestListener() *testListener {
	return &testListener{results: make(chan testAcceptResult, 1)}
}

func (l *testListener) Accept() (net.Conn, error) {
	result := <-l.results
	return result.connection, result.err
}

func (l *testListener) Close() error {
	select {
	case l.results <- testAcceptResult{err: net.ErrClosed}:
	default:
	}
	return nil
}

func (l *testListener) Addr() net.Addr {
	return testAddr("qtest")
}

type testAddr string

func (a testAddr) Network() string { return string(a) }
func (a testAddr) String() string  { return string(a) }
