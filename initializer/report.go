// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/edgelesssys/contrast/internal/atls/issuer"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/spf13/cobra"
)

// NewReportCmd creates a Cobra subcommand that prints a report to stdout.
func NewReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "write an attestation report to stdout",
		RunE:  runReport,
	}
	return cmd
}

func runReport(cmd *cobra.Command, _ []string) error {
	log, err := logger.Default()
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	issuer, err := issuer.New(log)
	if err != nil {
		return fmt.Errorf("creating issuer: %w", err)
	}
	quote, err := issuer.Issue(ctx, [64]byte{})
	if err != nil {
		return fmt.Errorf("creating report: %w", err)
	}

	encoder := base64.NewEncoder(base64.StdEncoding, os.Stdout)
	defer encoder.Close()

	if _, err := io.Copy(encoder, bytes.NewBuffer(quote)); err != nil {
		return fmt.Errorf("writing report to stdout: %w", err)
	}

	return nil
}
