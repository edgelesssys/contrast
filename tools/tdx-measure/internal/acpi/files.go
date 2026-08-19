// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package acpi handles QEMU ACPI fw_cfg blobs measured by OVMF.
package acpi

// TableLoader is the BIOS linker/loader fw_cfg path.
const TableLoader = "etc/table-loader"

// OVMFMeasuredFiles returns OVMF's fixed measurement order.
func OVMFMeasuredFiles() []string {
	return []string{
		TableLoader,
		"etc/acpi/rsdp",
		"etc/acpi/tables",
	}
}
