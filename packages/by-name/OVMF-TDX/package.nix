# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  edk2,
  nasm,
  acpica-tools,
  withACPIValidation ? true,
  debug ? false,
}:

edk2.mkDerivation "OvmfPkg/IntelTdx/IntelTdxX64.dsc" {
  name = "OVMF-TDX";

  buildFlags = [
    "-D BUILD_SHELL=FALSE" # We don't want any shell functionality compiled into the firmware.
    "-D BUILD_FIRMWARE_UI=FALSE" # We don't need any interactive firmware UI.
    "-D BUILD_SMBIOS=FALSE" # We don't need SMBIOS support, and the handoff table changes on every qemu update.
  ]
  ++ lib.optionals debug [ "-D DEBUG_ON_SERIAL_PORT=TRUE" ];

  buildConfig = if debug then "DEBUG" else "RELEASE";

  nativeBuildInputs = [
    nasm
    acpica-tools
  ];

  # When applying these patches with `git am`, use `--ignore-space-change`
  # to ignore CRLF conversion changes. When creating the patches, you should
  # still set your VSCode or editor to use CRLF line endings to match the
  # upstream style and create a sane diff.
  patches = [
    # Skip the measurement of the guest-memory-dependent TD HOBs and verify
    # them in the measured firmware instead.
    ./0001-TdxHelperLib-verify-Hobs-instead-of-measuring-them.patch
    # Add BUILD_FIRMWARE_UI toggle to disable the firmware UI to be included.
    ./0002-IntelTdxX64-add-toggle-to-disable-firmware-UI.patch
    # Add BUILD_SMBIOS toggle to disable SMBIOS support to be included.
    # SMBIOS handoff table changes on every QEMU update, and SMBIOS support isn't needed for the guest.
    ./0003-IntelTdx-add-toggle-to-disable-SMBIOS.patch
  ]
  ++ lib.optionals withACPIValidation [
    # Skip the measurement of the guest-memory and device-dependent ACPI tables and verify
    # them in the measured firmware instead.
    ./0004-QemuFwCfgAcpi-verify-ACPI-data-instead-of-measuring.patch

    # Skip the measurement of the non-critical `X-PciMmio64Mb` fw_cfg string that differs
    # based on the amount and BAR size of PCI devices (i.e. GPUs) being passed.
    # This isn't directly adjacent to the ACPI verification, but we still place it in
    # this branch, as it's only relevant for the TDX-GPU runtime as of now, which
    # uses `withACPIValidation`.
    ./0005-QemuFwCfgCacheInit-Skip-measuring-PCI-root-port-size.patch

    # Scan DSDT/SSDT AML bytecode for blocked device names (e.g. injected ACPI devices).
    ./0006-QemuFwCfgAcpi-add-AML-device-blocklist-verification.patch
    # Replace device name blocklist with SystemMemory OperationRegion allowlist.
    # Blocks any SystemMemory region not matching HPET (0xFED00000, 0x0400).
    ./0007-QemuFwCfgAcpi-replace-device-blocklist-with-SystemMe.patch
    # Block Load, LoadTable, and DataTableRegion opcodes to prevent
    # dynamic AML loading that bypasses static OperationRegion scanning.
    ./0008-QemuFwCfgAcpi-block-Load-LoadTable-and-DataTableRegi.patch
    # Re-verify blob data after all table-loader commands (ADD_POINTER etc.)
    # to close TOCTOU gap where pointer patching could modify verified AML.
    ./0009-QemuFwCfgAcpi-re-verify-blobs-after-pointer-patching.patch
  ];

  hardeningDisable = [
    "format"
    "stackprotector"
    "pic"
    "fortify"
  ];
}
