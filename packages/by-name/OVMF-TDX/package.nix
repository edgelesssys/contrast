# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  edk2,
  nasm,
  acpica-tools,
  withACPIVerificationInsecure ? false,
  debug ? true,
}:

edk2.mkDerivation "OvmfPkg/IntelTdx/IntelTdxX64.dsc" {
  name = "OVMF-TDX";

  buildFlags = lib.optionals debug [ "-D DEBUG_ON_SERIAL_PORT=TRUE" ];

  buildConfig = if debug then "DEBUG" else "RELEASE";

  nativeBuildInputs = [
    nasm
    acpica-tools
  ];

  # When applying these patches with `git am`, use `--ignore-space-change`
  # to ignore CRLF conversion changes. When creating the patches, you should
  # still set your VSCode or editor to use CRLF line endings to match the
  # upstream style and create a sane diff.
  # patches = [
  #   # Skip the measurement of the guest-memory-dependent TD HOBs and verify
  #   # them in the measured firmware instead.
  #   ./0001-TdxHelperLib-verify-Hobs-instead-of-measuring-them.patch
  #   # Make the measurement of the SMBIOS handoff table independent of the amount of memory.
  #   # The patch was necessary after the bump from edk2 202411 to 202508.01, as the SMBIOS
  #   # handoff table wasn't measured before.
  #   ./0002-SmbiosMeasurementDxe-filter-handoff-table.patch
  # ]
  # ++ lib.optionals withACPIVerificationInsecure [
  #   # Skip the measurement of the guest-memory and device-dependent ACPI tables and verify
  #   # them in the measured firmware instead.
  #   ./0003-QemuFwCfgAcpi-verify-ACPI-data-instead-of-measuring.patch

  #   ./0004-tdx-add-logging.patch
  # ];

  hardeningDisable = [
    "format"
    "stackprotector"
    "pic"
    "fortify"
  ];
}
