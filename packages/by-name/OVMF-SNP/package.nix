# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  edk2,
  fetchFromGitHub,
  nasm,
  acpica-tools,
}:

edk2.mkDerivation "OvmfPkg/AmdSev/AmdSevX64.dsc" {
  name = "OVMF-SNP";

  src = fetchFromGitHub {
    owner = "edgelesssys";
    repo = "ovmf";
    # https://github.com/edgelesssys/ovmf/commits/apic-mmio-fix4-edgeless/
    # which is https://github.com/AMDESE/ovmf/commits/apic-mmio-fix4
    # including https://github.com/tianocore/edk2/commit/95d8a1c255cfb8e063d679930d08ca6426eb5701.
    rev = "3c5968fd4e5fed316c3435bd266142dfc2d2840e";
    fetchSubmodules = true;
    hash = "sha256-0ijeEmBOhuBEvNxkGBsP/yUqWEzEXy928X0RHFl00d4=";
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
