# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  edk2,
  nasm,
  acpica-tools,
}:

edk2.mkDerivation "OvmfPkg/AmdSev/AmdSevX64.dsc" {
  name = "OVMF-SNP";

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
  patches = [
    # Skip the measurement of the guest-memory and device-dependent ACPI tables and verify
    # them in the measured firmware instead.
    ./0001-QemuFwCfgAcpi-verify-ACPI-data-instead-of-measuring.patch
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
