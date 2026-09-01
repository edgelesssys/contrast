// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/edgelesssys/contrast/tdx-measure/internal/acpi"
	"github.com/edgelesssys/contrast/tdx-measure/internal/qtest"
)

const (
	pciexbarOffset      = 0x60
	ecamBase            = 0xe0000000
	ecamEnable          = 0x1
	ich9LPCDevice       = 0x1f
	ich9LPCPMBaseOffset = 0x40
	ich9LPCPMBaseValue  = 0x0600
	ich9LPCACPIOffset   = 0x44
	ich9LPCACPIEnable   = 0x80
)

type dumpConfig struct {
	OutputDir  string
	QEMUBinary string
	QEMUArgs   []string
}

// dump extracts OVMF-measured ACPI blobs without running guest code.
func dump(ctx context.Context, config dumpConfig) (returnErr error) {
	if config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}
	if config.QEMUBinary == "" {
		return fmt.Errorf("QEMU binary is required")
	}
	if err := os.MkdirAll(config.OutputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	session, err := qtest.Start(ctx, config.QEMUBinary, config.QEMUArgs)
	if err != nil {
		return err
	}
	defer func() {
		if err := session.Close(); err != nil && returnErr == nil {
			returnErr = err
		}
	}()

	client := session.Client()
	if err := programACPIChipsetRegisters(ctx, client); err != nil {
		return fmt.Errorf("programming ACPI chipset registers: %w", err)
	}

	fwConfig, err := qtest.OpenFWConfig(ctx, client, acpi.TableLoader)
	if err != nil {
		return err
	}
	for _, name := range acpi.OVMFMeasuredFiles() {
		blob, err := fwConfig.ReadFile(ctx, name)
		if err != nil {
			return err
		}
		if err := acpi.WriteBlob(config.OutputDir, name, blob); err != nil {
			return err
		}
	}
	return nil
}

// programACPIChipsetRegisters reproduces OVMF's q35 ACPI register setup.
func programACPIChipsetRegisters(ctx context.Context, client *qtest.Client) error {
	if err := client.PCIConfigWrite32(ctx, 0, 0, 0, pciexbarOffset+4, ecamBase>>32); err != nil {
		return err
	}
	if err := client.PCIConfigWrite32(ctx, 0, 0, 0, pciexbarOffset, ecamBase|ecamEnable); err != nil {
		return err
	}
	if err := client.PCIConfigWrite32(ctx, 0, ich9LPCDevice, 0, ich9LPCPMBaseOffset, ich9LPCPMBaseValue); err != nil {
		return err
	}
	return client.PCIConfigWrite8(ctx, 0, ich9LPCDevice, 0, ich9LPCACPIOffset, ich9LPCACPIEnable)
}

func qemuVersion(ctx context.Context, binary string) (string, error) {
	output, err := exec.CommandContext(ctx, binary, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("reading QEMU version: %w", err)
	}
	version, _, _ := strings.Cut(string(output), "\n")
	if version == "" {
		return "", fmt.Errorf("QEMU returned an empty version")
	}
	return strings.TrimSuffix(version, "\r"), nil
}
