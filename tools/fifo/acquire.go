// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgelesssys/contrast/tools/fifo/internal/lease"
	"github.com/spf13/cobra"
)

func newAcquireCmd() *cobra.Command {
	var namespace, lockName string
	var timeout, duration time.Duration
	var verbose bool

	cmd := &cobra.Command{
		Use:   "acquire",
		Short: "Acquire the named lock, blocking until it is available",
		Long: `Acquire the named lock in FIFO order.

The command blocks until the lock is available and this caller is at the
front of the queue. On success it prints the holder identity to stdout;
save this value to pass to "fifo release" later. On error, the command
returns a non-zero exit code.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAcquire(cmd, namespace, lockName, timeout, duration, verbose)
		},
	}
	cmd.Flags().StringVar(&namespace, "namespace", defaultNamespace, "Kubernetes namespace for the Lease objects")
	cmd.Flags().StringVar(&lockName, "lease", defaultLeaseName, "Name of the main Lease object")
	cmd.Flags().DurationVar(&timeout, "timeout", 65*time.Minute, "Maximum waiting time to acquire the Lease object.")
	cmd.Flags().DurationVar(&duration, "duration", 60*time.Minute, "Maximum validity of the Lease object.")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable debug output")
	return cmd
}

func runAcquire(cmd *cobra.Command, namespace, lockName string, timeout, leaseDuration time.Duration, verbose bool) error {
	client, err := newLeaseClient(namespace)
	if err != nil {
		return fmt.Errorf("creating k8s client: %w", err)
	}

	logOpts := &slog.HandlerOptions{}
	if verbose {
		logOpts.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), logOpts))

	host, _ := os.Hostname()
	if host == "" {
		host = "unknown-host.invalid"
	}
	holderID := fmt.Sprintf("%s/%d/%08x", host, os.Getpid(), rand.Uint32())
	lease := lease.New(lockName, holderID, leaseDuration, client, logger)

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := lease.Acquire(ctx); err != nil {
		return fmt.Errorf("acquiring lease: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", holderID); err != nil {
		return fmt.Errorf("writing to stdout: %w", err)
	}
	return nil
}
