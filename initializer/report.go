// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os/signal"
	"strings"
	"syscall"

	"github.com/edgelesssys/contrast/internal/atls/issuer"
	"github.com/edgelesssys/contrast/internal/attestation/insecure"
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
	cmd.Flags().Var(&diagnostics{}, "diagnostics", "which diagnostics to print")
	return cmd
}

func runReport(cmd *cobra.Command, _ []string) (retErr error) {
	log, err := logger.Default()
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}

	diagnostics, ok := cmd.Flags().Lookup("diagnostics").Value.(*diagnostics)
	if !ok {
		panic("programming error, unreachable under normal circumstances")
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	issuer, err := issuer.New(log, getCollateralProxy(log))
	if err != nil {
		runChecks(log, diagnostics)
		return fmt.Errorf("creating issuer: %w", err)
	}
	if _, ok := issuer.(*insecure.Issuer); ok {
		runChecks(log, diagnostics)
		return errors.New("no acceptable issuer found (issuer.New fell back to insecure)")
	}
	quote, err := issuer.Issue(ctx, [64]byte{})
	if err != nil {
		runChecks(log, diagnostics)
		return fmt.Errorf("creating report: %w", err)
	}

	if _, err := fmt.Println(base64.StdEncoding.EncodeToString(quote)); err != nil {
		return fmt.Errorf("writing report to stdout: %w", err)
	}
	return nil
}

func runChecks(logger *slog.Logger, d *diagnostics) {
	if !d.noSNP {
		logger.Info("=== Running SEV-SNP compatibility checks ===")
		runOpenCheck(logger, "/dev/sev-guest")
		runTSMCheck(logger, "sev_guest")
	}

	if !d.noTDX {
		logger.Info("=== Running TDX compatibility checks ===")
		runOpenCheck(logger, "/dev/tdx_guest")
		runTSMCheck(logger, "tdx_guest")
	}
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
	defer func() {
		if err := r.Destroy(); err != nil {
			logger.Error("failed to destroy configfs-tsm report", "error", err)
		}
	}()
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

var supportedDiagnosticsFlags = []string{"all", "snp", "tdx", "none"}

// diagnostics knows which diagnostics should be run if issuance failed.
type diagnostics struct {
	// Inverted to make "all" the default.
	noSNP, noTDX bool
}

// Set implements [pflag.Value].
func (d *diagnostics) Set(v string) error {
	// All branches set both fields such that the last flag overrides previous flags.
	switch strings.ToLower(v) {
	case "snp":
		d.noTDX = true
		d.noSNP = false
	case "tdx":
		d.noSNP = true
		d.noTDX = false
	case "none":
		d.noSNP = true
		d.noTDX = true
	case "all":
		d.noSNP = false
		d.noTDX = false
	default:
		// cobra already reflects v in the printed output, we only need to add what is acceptable.
		return fmt.Errorf("expected one of %v", supportedDiagnosticsFlags)
	}
	return nil
}

// String implements [pflag.Value] and is called to print the default value.
func (d *diagnostics) String() string {
	return "all"
}

// Type implements [pflag.Value] and is called to render the flag argument for --help.
func (d *diagnostics) Type() string {
	return strings.Join(supportedDiagnosticsFlags, "|")
}
