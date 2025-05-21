// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/edgelesssys/contrast/tdx-measure/rtmr"
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
		newRtMrCmd(),
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
		Args: cobra.NoArgs,
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

// newRtMrCmd creates the tdx-measure rtmr subcommand.
func newRtMrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rtmr -f OVMF.fd -k bzImage [0|1|2|3]",
		Short: "calculate the RTMR for a firmware and kernel file",
		Long: `Calculate the RTMR for a firmware and kernel file.

		This will parse the firmware according to the TDX Virtual Firmware Design Guide
		and/or hash the kernel and pre-calculate a given RTMR.`,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"0", "1", "2", "3"},
		RunE:      runRtMr,
	}
	cmd.Flags().StringP("firmware", "f", "OVMF.fd", "path to firmware file")
	if err := cmd.MarkFlagFilename("firmware", "fd"); err != nil {
		panic(err)
	}
	cmd.Flags().StringP("kernel", "k", "bzImage", "path to kernel file")
	if err := cmd.MarkFlagFilename("kernel"); err != nil {
		panic(err)
	}
	cmd.Flags().StringP("initrd", "i", "initrd.zst", "path to initrd file")
	if err := cmd.MarkFlagFilename("initrd"); err != nil {
		panic(err)
	}
	cmd.Flags().StringP("cmdline", "c", "", "kernel command line")
	return cmd
}

func runRtMr(cmd *cobra.Command, args []string) error {
	var digest [48]byte
	switch args[0] {
	case "0":
		firmwarePath, err := cmd.Flags().GetString("firmware")
		if err != nil {
			return err
		}
		firmware, err := os.ReadFile(firmwarePath)
		if err != nil {
			return fmt.Errorf("can't read firmware file: %w", err)
		}

		digest, err = rtmr.CalcRtmr0(firmware)
		if err != nil {
			return fmt.Errorf("can't calculate RTMR 0: %w", err)
		}
	case "1":
		kernelPath, err := cmd.Flags().GetString("kernel")
		if err != nil {
			return err
		}
		kernel, err := os.ReadFile(kernelPath)
		if err != nil {
			return fmt.Errorf("can't read kernel file: %w", err)
		}
		initrdPath, err := cmd.Flags().GetString("initrd")
		if err != nil {
			return err
		}
		initrd, err := os.ReadFile(initrdPath)
		if err != nil {
			return fmt.Errorf("can't read initrd file: %w", err)
		}
		digest, err = rtmr.CalcRtmr1(kernel, initrd)
		if err != nil {
			return fmt.Errorf("can't calculate RTMR 1: %w", err)
		}
	case "2":
		cmdLine, err := cmd.Flags().GetString("cmdline")
		if err != nil {
			return err
		}
		initrdPath, err := cmd.Flags().GetString("initrd")
		if err != nil {
			return err
		}
		initrd, err := os.ReadFile(initrdPath)
		if err != nil {
			return fmt.Errorf("can't read initrd file: %w", err)
		}
		digest, err = rtmr.CalcRtmr2(cmdLine, initrd)
		if err != nil {
			return fmt.Errorf("can't calculate RTMR 2: %w", err)
		}
	case "3":
		digest = [48]byte{}
	}

	hexDigest := hex.EncodeToString(digest[:])
	fmt.Print(hexDigest)

	return nil
}
