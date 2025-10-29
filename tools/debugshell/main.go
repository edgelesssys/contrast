// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
)

var bashPath = "/bin/sh" // Path is swapped out during package build, /bin/sh for local development.

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Check that the trigger file exists. If it does not, something is wrong with setup,
	// as the systemd unit for debug shell should only be triggered if the file exists.
	const triggerFile = "/run/measured-cfg/contrast.insecure-debug"
	if _, err := os.Stat(triggerFile); os.IsNotExist(err) {
		log.Fatalf("Debug shell trigger file %s does not exist, refusing to start debug shell server", triggerFile)
	} else if err != nil {
		log.Fatalf("Error checking debug shell trigger file %s: %v", triggerFile, err)
	}

	s := &ssh.Server{
		Addr:    "127.0.0.1:2222",
		Handler: handle,
		PasswordHandler: func(_ ssh.Context, _ string) bool {
			return true // Allow all passwords (insecure!)
		},
		PublicKeyHandler: func(_ ssh.Context, _ ssh.PublicKey) bool {
			return true // Allow all keys (insecure!)
		},
	}

	wg := sync.WaitGroup{}

	wg.Go(func() {
		defer wg.Done()
		log.Printf("Starting debug shell server on %s", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			log.Fatalf("Error: %v", err)
		}
	})

	wg.Go(func() {
		defer wg.Done()
		<-ctx.Done()
		if err := s.Close(); err != nil {
			log.Printf("Error closing SSH server: %v\n", err)
		}
	})

	wg.Wait()
	log.Println("Debug shell server stopped")
}

// This handler is called after the SSH handshake,
// after client requested a shell or exec.
func handle(s ssh.Session) {
	log.Printf("Handling new session for user %s from %s", s.User(), s.RemoteAddr())

	ptyReq, winCh, isPty := s.Pty()
	if !isPty {
		if len(s.Command()) == 0 {
			log.Printf("No pty or command requested from %s", s.RemoteAddr())
			fmt.Fprintln(s, "Error: no pty or command requested")
			_ = s.Exit(1)
			return
		}

		log.Printf("Executing non-interactive command for %s", s.RemoteAddr())
		cmd := exec.CommandContext(s.Context(), bashPath, "-lc", s.RawCommand())
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")
		cmd.Stdout = s
		cmd.Stderr = s
		var exitErr *exec.ExitError
		if err := cmd.Run(); errors.As(err, &exitErr) {
			_ = s.Exit(exitErr.ExitCode())
		} else if err != nil {
			fmt.Fprintf(s, "cmd run error: %v\n", err)
			_ = s.Exit(1)
		}
		_ = s.Exit(0)
		return
	}

	// Start bash (interactive login) in pty
	cmd := exec.CommandContext(s.Context(), bashPath, "-li")
	cmd.Env = append(os.Environ(), "TERM="+ptyReq.Term)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(s, "failed to start pty: %v\n", err)
		return
	}
	defer ptmx.Close()

	// Set initial window size from ptyReq
	_ = pty.Setsize(ptmx, &pty.Winsize{
		Rows: uint16(ptyReq.Window.Height),
		Cols: uint16(ptyReq.Window.Width),
	})

	// Monitor window-change requests
	go func() {
		for win := range winCh {
			_ = pty.Setsize(ptmx, &pty.Winsize{
				Rows: uint16(win.Height),
				Cols: uint16(win.Width),
			})
			// send SIGWINCH to the process so it updates children
			_ = cmd.Process.Signal(syscall.SIGWINCH)
		}
	}()

	// Copy between the pty and SSH session
	go func() {
		_, _ = io.Copy(ptmx, s) // session → pty (stdin)
	}()
	// Block on output copying. If ptmx closes its output, it won't accept input anyway.
	_, _ = io.Copy(s, ptmx) // pty → session (stdout & stderr)

	// On exit, send HUP
	_ = cmd.Process.Signal(syscall.SIGHUP)
	log.Printf("Session for user %s from %s closed", s.User(), s.RemoteAddr())
}
