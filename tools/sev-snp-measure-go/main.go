// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// sev-snp-measure computes the AMD SEV-SNP launch digest for a guest firmware image.
// This is a Go reimplementation of the Python sev-snp-measure tool, covering the
// subset of flags required for Kata Containers SNP measurement (--mode snp, VMM QEMU).
package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/edgelesssys/contrast/internal/snp"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		ovmfPath      string
		kernelPath    string
		initrdPath    string
		appendStr     string
		vcpus         int
		vcpuType      string
		vcpuSig       uint32
		guestFeatures uint64
		outputFormat  string
	)

	cmd := &cobra.Command{
		Use:   "sev-snp-measure",
		Short: "Calculate AMD SEV-SNP guest launch measurement",
		Long: `sev-snp-measure calculates the AMD SEV-SNP launch digest for a guest.

Only --mode snp with VMM type QEMU is supported (the minimum required for
Kata Containers SNP measurement). Output is the hex-encoded SHA-384 digest.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Resolve vCPU signature.
			var resolvedSig uint32
			switch {
			case vcpuSig != 0:
				resolvedSig = vcpuSig
			case vcpuType != "":
				sig, err := snp.LookupCPUSig(vcpuType)
				if err != nil {
					return err
				}
				resolvedSig = sig
			default:
				return fmt.Errorf("--vcpu-type or --vcpu-sig is required")
			}

			if vcpus <= 0 {
				return fmt.Errorf("--vcpus must be a positive integer")
			}
			if ovmfPath == "" {
				return fmt.Errorf("--ovmf is required")
			}

			digest, err := snp.CalcSNPLaunchDigest(
				ovmfPath, vcpus, resolvedSig,
				kernelPath, initrdPath, appendStr,
				guestFeatures,
			)
			if err != nil {
				return err
			}

			switch outputFormat {
			case "hex":
				fmt.Println(hex.EncodeToString(digest[:]))
			default:
				return fmt.Errorf("unsupported output format %q (only \"hex\" is supported)", outputFormat)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&ovmfPath, "ovmf", "", "Path to OVMF firmware binary (required)")
	f.StringVar(&kernelPath, "kernel", "", "Path to kernel bzImage")
	f.StringVar(&initrdPath, "initrd", "", "Path to initrd (use with --kernel)")
	f.StringVar(&appendStr, "append", "", "Kernel command line (use with --kernel)")
	f.IntVar(&vcpus, "vcpus", 0, "Number of guest vCPUs (required)")
	f.StringVar(&vcpuType, "vcpu-type", "", "vCPU type (e.g. EPYC-Milan, EPYC-Genoa)")
	f.Uint32Var(&vcpuSig, "vcpu-sig", 0, "vCPU CPUID signature value (alternative to --vcpu-type)")
	f.Uint64Var(&guestFeatures, "guest-features", 0x1, "Guest feature flags (hex, e.g. 0x1)")
	f.StringVar(&outputFormat, "output-format", "hex", "Output format: hex")

	// --mode is accepted for compatibility with the Python tool but must be "snp".
	var mode string
	f.StringVar(&mode, "mode", "snp", "Guest mode (only \"snp\" is supported)")
	cmd.PreRunE = func(_ *cobra.Command, _ []string) error {
		if strings.ToLower(mode) != "snp" {
			return fmt.Errorf("only --mode snp is supported (got %q)", mode)
		}
		return nil
	}

	cmd.AddCommand(newAPEIPCmd())

	return cmd
}

func newAPEIPCmd() *cobra.Command {
	var ovmfPath string

	cmd := &cobra.Command{
		Use:   "ap-eip",
		Short: "Print the SEV-ES AP reset EIP from an OVMF firmware image",
		Long: `ap-eip reads the SEV-ES AP reset EIP from the OVMF footer table and prints
it as an 8-digit lowercase hex value. This value is specific to the OVMF build
and is needed to calculate launch measurements for varying vCPU counts at
verify time without storing per-vCPU measurements.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if ovmfPath == "" {
				return fmt.Errorf("--ovmf is required")
			}
			ovmf, err := snp.NewOVMF(ovmfPath)
			if err != nil {
				return fmt.Errorf("parsing OVMF: %w", err)
			}
			apEIP, err := ovmf.SEVESResetEIP()
			if err != nil {
				return fmt.Errorf("reading OVMF reset EIP: %w", err)
			}
			fmt.Printf("%08x\n", apEIP)
			return nil
		},
	}

	cmd.Flags().StringVar(&ovmfPath, "ovmf", "", "Path to OVMF firmware binary (required)")

	return cmd
}
