// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgelesssys/contrast/tools/fifo/internal/lease"
	"github.com/spf13/cobra"
)

func newReleaseCmd() *cobra.Command {
	var namespace, leaseName string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "release <holder-id>",
		Short: "Release the named lock",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRelease(cmd, namespace, leaseName, args[0], verbose)
		},
	}
	cmd.Flags().StringVar(&namespace, "namespace", defaultNamespace, "Kubernetes namespace for the Lease objects")
	cmd.Flags().StringVar(&leaseName, "lease", defaultLeaseName, "name of the Lease")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable debug output")
	return cmd
}

func runRelease(cmd *cobra.Command, namespace, leaseName, holderID string, verbose bool) error {
	client, err := newLeaseClient(namespace)
	if err != nil {
		return fmt.Errorf("creating k8s client: %w", err)
	}

	logOpts := &slog.HandlerOptions{}
	if verbose {
		logOpts.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), logOpts))

	// This is not relevant for releasing, set any valid duration.
	leaseDuration := time.Second
	lease := lease.New(leaseName, holderID, leaseDuration, client, logger)

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	return lease.Release(ctx)
}
