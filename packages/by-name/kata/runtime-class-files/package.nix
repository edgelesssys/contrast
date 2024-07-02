# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ stdenvNoCC
, kata
, fetchurl
, OVMF
}:

let
  image = kata.kata-image;
  kernel = "${kata.kata-kernel-uvm}/bzImage";

  # TODO(msanft): building a non-NixOS QEMU is hard, investigate later and pin it for now.
  # Binary is sourced from: https://launchpad.net/ubuntu/+archive/primary/+sourcefiles/qemu/1:8.2.2+ds-0ubuntu1/qemu_8.2.2+ds.orig.tar.xz
  qemu-bin = fetchurl {
    url = "https://cdn.confidential.cloud/contrast/node-components/1718800762/qemu-system-x86_64";
    hash = "sha256-7MS/tK6q4D8y/FH6VcfARQLhIuvtNP6TsGfy+0o9kSc=";
  };

  ovmf-code = OVMF.firmware;
  ovmf-vars = OVMF.variables;

  containerd-shim-contrast-cc-v2 = "${kata.kata-runtime}/bin/containerd-shim-kata-v2";
in

stdenvNoCC.mkDerivation {
  name = "runtime-class-files";
  inherit (kata.kata-image) version;

  dontUnpack = true;

  # TODO(msanft): perform the actual launch digest calculation.
  buildPhase = ''
    mkdir -p $out
    sha256sum ${image} ${kernel} ${qemu-bin} ${containerd-shim-contrast-cc-v2} ${ovmf-code} ${ovmf-vars} | sha256sum | cut -d " " -f 1 > $out/launch-digest.hex
    printf "contrast-cc-%s" "$(cat $out/launch-digest.hex | head -c 32)" > $out/runtime-handler
  '';

  passthru = {
    inherit kernel image qemu-bin containerd-shim-contrast-cc-v2 ovmf-code ovmf-vars;
  };
}
