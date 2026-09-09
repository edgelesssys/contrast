# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  qemuACPIBlobs,
  tdx-measure,
}:

tdx-measure.overrideAttrs (_old: {
  pname = "qemu-acpi-blobs-test";
  ACPI_BLOBS_DEFAULT_DIR = qemuACPIBlobs {
    vcpus = 1;
  };
  ACPI_BLOBS_LEGACY_SERIAL_DIR = qemuACPIBlobs {
    vcpus = 1;
    legacySerial = true;
  };
  doCheck = true;
})
