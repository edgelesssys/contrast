# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  edk2,
  fetchFromGitHub,
  nasm,
  acpica-tools,
}:

edk2.mkDerivation "OvmfPkg/AmdSev/AmdSevX64.dsc" rec {
  name = "OVMF-SNP";
  src = fetchFromGitHub {
    owner = "amdese";
    repo = "ovmf";
    # https://github.com/AMDESE/ovmf/tree/apic-mmio-fix4
    rev = "64b3116ed087f8cb026201e56e42efe751e2cf7d";
    fetchSubmodules = true;
    hash = "sha256-nb4p01Y+M5a3EEJb9692hcFkU7HgpbG1rZa60T+I8N4=";
  };
  postPatch = ''
    touch OvmfPkg/AmdSev/Grub/grub.efi
  '';
  # Disable making all warnings errors. Nix's GCC is fairly new, so it spews a
  # few more warnings, but that shouldn't prevent us from building OVMF.
  postConfigure = ''
    sed -i "s/-Werror//g" Conf/tools_def.txt
  '';

  nativeBuildInputs = [
    nasm
    acpica-tools
  ];

  hardeningDisable = [
    "format"
    "stackprotector"
    "pic"
    "fortify"
  ];
}
