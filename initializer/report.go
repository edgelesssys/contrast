// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/edgelesssys/contrast/internal/atls/issuer"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/google/go-configfs-tsm/configfs/linuxtsm"
	"github.com/google/go-configfs-tsm/report"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
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

func runReport(cmd *cobra.Command, _ []string) (retErr error) {
	log, err := logger.Default()
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}

	defer func() {
		if retErr == nil {
			return
		}
		runChecks(log)
	}()

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	issuer, err := issuer.New(log, getCollateralProxy(log))
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

func runChecks(logger *slog.Logger) {
	logger.Info("=== Running SEV-SNP compatibility checks ===")
	runOpenCheck(logger, "/dev/sev-guest")
	runTSMCheck(logger, "sev_guest")

	logger.Info("=== Running TDX compatibility checks ===")
	runOpenCheck(logger, "/dev/tdx_guest")
	runTSMCheck(logger, "tdx_guest")
}

func runOpenCheck(logger *slog.Logger, name string) {
	fd, err := unix.Open(name, unix.O_RDWR, 0)
	if err != nil {
		logger.Error("can't open device", "device", name, "error", err)
	} else {
		_ = unix.Close(fd)
		logger.Info("can open device", "device", name)
	}
}

func runTSMCheck(logger *slog.Logger, name string) {
	c, err := linuxtsm.MakeClient()
	if err != nil {
		logger.Error("can't create configfs client", "error", err)
		return
	}
	r, err := report.Create(c, &report.Request{})
	if err != nil {
		logger.Error("can't create report", "error", err)
		return
	}
	provider, err := r.ReadOption("provider")
	if err != nil {
		logger.Error("can't read report provider", "error", err)
		return
	}
	if string(provider) != name+"\n" {
		logger.Error("unexpected report provider", "got", string(provider), "want", name+"\n")
	} else {
		logger.Info("configfs provider supported")
	}
}
