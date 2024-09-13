// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/edgelesssys/contrast/tdx-measure/tdvf"
	"github.com/spf13/cobra"
)

var version = "0.0.0-dev"

func main() {
	if err := execute(); err != nil {
		os.Exit(1)
	}
}

func execute() error {
	cmd := newRootCmd()
	return cmd.ExecuteContext(context.Background())
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Version: version,
		Use:     "tdx-measure",
		Short:   "tdx-measure",
	}
	root.SetOut(os.Stdout)

	root.InitDefaultVersionFlag()
	root.AddCommand(
		newMrTdCmd(),
	)

	return root
}

// newMrTdCmd creates the tdx-measure mrtd subcommand.
func newMrTdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mrtd -f OVMF.fd",
		Short: "calculate the MRTD for a firmware file",
		Long: `Calculate the MRTD for a firmware file.

		This will parse the firmware according to the TDX Virtual Firmware Design Guide and
		pre-calculate the MRTD.`,
		RunE: runMrTd,
	}
	cmd.Flags().StringP("firmware", "f", "OVMF.fd", "path to firmware file")
	if err := cmd.MarkFlagFilename("firmware", "fd"); err != nil {
		panic(err)
	}
	return cmd
}

func runMrTd(cmd *cobra.Command, _ []string) error {
	firmwarePath, err := cmd.Flags().GetString("firmware")
	if err != nil {
		return fmt.Errorf("can't get firmware arg: %w", err)
	}

	firmware, err := os.ReadFile(firmwarePath)
	if err != nil {
		return fmt.Errorf("can't read firmware file: %w", err)
	}

	digest, err := tdvf.CalculateMrTd(firmware)
	if err != nil {
		return fmt.Errorf("can't calculate MRTD for firmware: %w", err)
	}

	hexDigest := hex.EncodeToString(digest[:])
	fmt.Print(hexDigest)

	return nil
}
