# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  edk2,
  lib,
  nasm,
  acpica-tools,
  debug ? false,
  withACPIValidation ? true,
}:

edk2.mkDerivation "OvmfPkg/AmdSev/AmdSevX64.dsc" {
  name = "OVMF-SNP";

  buildFlags = lib.optionals debug [ "-D DEBUG_ON_SERIAL_PORT=TRUE" ];
  buildConfig = if debug then "DEBUG" else "RELEASE";

  postPatch = ''
    touch OvmfPkg/AmdSev/Grub/grub.efi
  '';

  postConfigure = ''
    # Disable making all warnings errors. Nix's GCC is fairly new, so it spews a
    # few more warnings, but that shouldn't prevent us from building OVMF.
    sed -i "s/-Werror//g" Conf/tools_def.txt

    # Disable the stack protection manually. We can't use `hardeningDisable` as it gets
    # overriden by the GCC flags in the EDK2 build system. (See Conf/tools_def.txt)
    sed -i "s/-fstack-protector/-fno-stack-protector/g" Conf/tools_def.txt
  '';

  # When applying these patches with `git am`, use `--ignore-space-change`
  # to ignore CRLF conversion changes. When creating the patches, you should
  # still set your VSCode or editor to use CRLF line endings to match the
  # upstream style and create a sane diff.
  patches =
    [
      # Skip the measurement of the guest-memory and device-dependent ACPI tables and verify
      # them in the measured firmware instead.
      ./0001-QemuFwCfgAcpi-verify-ACPI-data-instead-of-measuring.patch
    ]
    ++ lib.optionals withACPIValidation [
      # Scan DSDT/SSDT AML bytecode for blocked device names (e.g. injected ACPI devices).
      ./0002-QemuFwCfgAcpi-add-AML-device-blocklist-verification.patch
      # Replace device name blocklist with SystemMemory OperationRegion allowlist.
      # Blocks any SystemMemory region not matching HPET (0xFED00000, 0x0400).
      ./0003-QemuFwCfgAcpi-replace-device-blocklist-with-SystemMe.patch
      # Block Load, LoadTable, and DataTableRegion opcodes to prevent
      # dynamic AML loading that bypasses static OperationRegion scanning.
      ./0004-QemuFwCfgAcpi-block-Load-LoadTable-and-DataTableRegi.patch
      # Re-verify blob data after all table-loader commands (ADD_POINTER etc.)
      # to close TOCTOU gap where pointer patching could modify verified AML.
      ./0005-QemuFwCfgAcpi-re-verify-blobs-after-pointer-patching.patch
    ];

  nativeBuildInputs = [
    nasm
    acpica-tools
  ];

  hardeningDisable = [
    "format"
    "pic"
    "fortify"
  ];
}
