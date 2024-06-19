# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ stdenvNoCC
, kata
, qemu
}:

let
  image = kata.kata-image;
  kernel = "${kata.kata-kernel-uvm}/bzImage";
  qemu-bin = "${qemu}/bin/qemu-system-x86_64";
  containerd-shim-contrast-cc-v2 = "${kata.kata-runtime}/bin/containerd-shim-kata-v2";
in

stdenvNoCC.mkDerivation {
  name = "runtime-class-files";
  version = "1718800762";

  dontUnpack = true;

  # TODO(msanft): perform the actual launch digest calculation.
  buildPhase = ''
    mkdir -p $out
    sha256sum ${image} ${kernel} ${qemu-bin} ${containerd-shim-contrast-cc-v2} | sha256sum | cut -d " " -f 1 > $out/launch-digest.hex
    echo -n "contrast-cc-" > $out/runtime-handler
    cat $out/launch-digest.hex | head -c 32 >> $out/runtime-handler
  '';

  passthru = {
    inherit kernel image qemu-bin containerd-shim-contrast-cc-v2;
  };
}
