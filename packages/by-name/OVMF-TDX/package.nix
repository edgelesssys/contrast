# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  edk2,
  nasm,
  acpica-tools,
}:

edk2.mkDerivation "OvmfPkg/IntelTdx/IntelTdxX64.dsc" rec {
  name = "OVMF-TDX";

  nativeBuildInputs = [
    nasm
    acpica-tools
  ];

  patches = [
    # Make the RTMR[0] measurement independent of the amount of memory.
    ./0001-verify-Hobs-instead-of-measuring-them.patch
  ];

  hardeningDisable = [
    "format"
    "stackprotector"
    "pic"
    "fortify"
  ];
}
