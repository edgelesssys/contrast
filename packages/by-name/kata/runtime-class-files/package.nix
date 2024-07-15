# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  stdenvNoCC,
  kata,
  OVMF-SNP,
  debugRuntime ? false,
  qemu-static,
}:

let
  image = kata.kata-image;
  kernel = "${kata.kata-kernel-uvm}/bzImage";

  qemu-bin = "${qemu-static}/bin/qemu-system-x86_64";
  qemu-share = "${qemu-static}/share/qemu";

  ovmf = "${OVMF-SNP}/FV/OVMF.fd";

  containerd-shim-contrast-cc-v2 = "${kata.kata-runtime}/bin/containerd-shim-kata-v2";

  kata-runtime = "${kata.kata-runtime}/bin/kata-runtime";
in

stdenvNoCC.mkDerivation {
  name = "runtime-class-files";
  inherit (kata.kata-image) version;

  dontUnpack = true;

  # TODO(msanft): perform the actual launch digest calculation.
  buildPhase = ''
    mkdir -p $out
    sha256sum ${image} ${kernel} ${qemu-bin} ${qemu-share}/kvmvapic.bin ${qemu-share}/linuxboot_dma.bin ${qemu-share}/efi-virtio.rom ${containerd-shim-contrast-cc-v2} ${ovmf} | sha256sum | cut -d " " -f 1 > $out/launch-digest.hex
    printf "contrast-cc-%s" "$(cat $out/launch-digest.hex | head -c 32)" > $out/runtime-handler
  '';

  passthru = {
    inherit
      kernel
      image
      qemu-bin
      qemu-share
      containerd-shim-contrast-cc-v2
      ovmf
      kata-runtime
      debugRuntime
      ;
  };
}
