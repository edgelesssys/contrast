# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ stdenvNoCC
, kata
, fetchurl
}:

let
  image = kata.kata-image;
  kernel = "${kata.kata-kernel-uvm}/bzImage";

  # TODO(msanft): building a non-NixOS QEMU is hard, investigate later and pin it for now.
  qemu-bin = fetchurl {
    url = "https://cdn.confidential.cloud/contrast/node-components/1718800762/qemu-system-x86_64";
    sha256 = "sha256-7MS/tK6q4D8y/FH6VcfARQLhIuvtNP6TsGfy+0o9kSc=";
  };

  containerd-shim-contrast-cc-v2 = "${kata.kata-runtime}/bin/containerd-shim-kata-v2";
in

stdenvNoCC.mkDerivation {
  name = "runtime-class-files";
  inherit (kata.kata-image) version;

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
