# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  edk2,
  nasm,
  acpica-tools,
  debug ? false,
}:

edk2.mkDerivation "OvmfPkg/IntelTdx/IntelTdxX64.dsc" {
  name = "OVMF-TDX";

  buildFlags = lib.optionals debug [ "-D DEBUG_ON_SERIAL_PORT=TRUE" ];

  buildConfig = if debug then "DEBUG" else "RELEASE";

  nativeBuildInputs = [
    nasm
    acpica-tools
  ];

  patches = [
    # Make the RTMR[0] measurement independent of the amount of memory.
    ./0001-verify-Hobs-instead-of-measuring-them.patch

    ./0002-don-t-measure-fw-cfg.patch
  ];

  hardeningDisable = [
    "format"
    "stackprotector"
    "pic"
    "fortify"
  ];
}
