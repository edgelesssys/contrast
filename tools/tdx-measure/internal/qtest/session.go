// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package qtest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// Session owns a QEMU process and its qtest connection.
type Session struct {
	client       *Client
	connection   net.Conn
	listener     net.Listener
	process      *os.Process
	processDone  <-chan error
	temporaryDir string
}

// Start launches QEMU with a qtest control socket and waits for QEMU to connect.
func Start(ctx context.Context, binary string, arguments []string) (*Session, error) {
	if binary == "" {
		return nil, fmt.Errorf("QEMU binary is required")
	}
	temporaryDir, err := os.MkdirTemp("", "qtest-")
	if err != nil {
		return nil, fmt.Errorf("creating qtest temporary directory: %w", err)
	}

	socketPath := filepath.Join(temporaryDir, "qtest.sock")
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "unix", socketPath)
	if err != nil {
		_ = os.RemoveAll(temporaryDir)
		return nil, fmt.Errorf("listening on qtest socket: %w", err)
	}

	qemuArguments := append([]string{}, arguments...)
	qemuArguments = append(
		qemuArguments,
		"-qtest", "unix:path="+socketPath,
		"-qtest-log", "none",
		"-display", "none",
	)
	command := exec.CommandContext(ctx, binary, qemuArguments...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		_ = listener.Close()
		_ = os.RemoveAll(temporaryDir)
		return nil, fmt.Errorf("starting QEMU: %w", err)
	}

	processDone := make(chan error, 1)
	go func() {
		processDone <- command.Wait()
		close(processDone)
	}()

	connection, err := acceptConnection(ctx, listener, processDone)
	if err != nil {
		_ = listener.Close()
		stopErr := stopProcess(command.Process, processDone)
		_ = os.RemoveAll(temporaryDir)
		return nil, errors.Join(err, stopErr)
	}

	return &Session{
		client:       NewClient(connection),
		connection:   connection,
		listener:     listener,
		process:      command.Process,
		processDone:  processDone,
		temporaryDir: temporaryDir,
	}, nil
}

// Client returns the qtest protocol client for the session.
func (s *Session) Client() *Client {
	return s.client
}

// Close terminates QEMU and releases the qtest socket and temporary directory.
func (s *Session) Close() error {
	_ = s.connection.Close()
	stopErr := stopProcess(s.process, s.processDone)
	_ = s.listener.Close()
	_ = os.RemoveAll(s.temporaryDir)
	return stopErr
}

func acceptConnection(ctx context.Context, listener net.Listener, processDone <-chan error) (net.Conn, error) {
	type acceptResult struct {
		connection net.Conn
		err        error
	}
	accepted := make(chan acceptResult, 1)
	go func() {
		connection, err := listener.Accept()
		accepted <- acceptResult{connection: connection, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for QEMU qtest connection: %w", ctx.Err())
	case err := <-processDone:
		if err == nil {
			return nil, errors.New("QEMU exited before connecting to qtest")
		}
		return nil, fmt.Errorf("QEMU exited before connecting to qtest: %w", err)
	case result := <-accepted:
		if result.err != nil {
			return nil, fmt.Errorf("accepting QEMU qtest connection: %w", result.err)
		}
		return result.connection, nil
	}
}

func stopProcess(process *os.Process, processDone <-chan error) error {
	if err := process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("terminating QEMU: %w", err)
	}
	select {
	case <-processDone:
		return nil
	case <-time.After(10 * time.Second):
		if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("killing QEMU after timeout: %w", err)
		}
		<-processDone
		return nil
	}
}
