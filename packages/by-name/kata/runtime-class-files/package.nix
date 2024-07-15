# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  stdenvNoCC,
  kata,
  sev-ovmf,
  debugRuntime ? false,
  qemu-static,
}:

let
  image = kata.kata-image;
  kernel = "${kata.kata-kernel-uvm}/bzImage";

  qemu-bin = "${qemu-static.override { snpSupport = true; }}/bin/qemu-system-x86_64";
  qemu-share = "${qemu-static.override { snpSupport = true; }}/share/qemu";

  ovmf = "${sev-ovmf}/FV/OVMF.fd";

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
    sha256sum ${image} ${kernel} ${qemu-bin} ${containerd-shim-contrast-cc-v2} ${ovmf} | sha256sum | cut -d " " -f 1 > $out/launch-digest.hex
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
