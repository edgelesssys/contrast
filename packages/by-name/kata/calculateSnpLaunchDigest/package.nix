# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  stdenvNoCC,
  kata,
  OVMF-SNP,
  python3Packages,
}:

{
  os-image,
  debug,
  vcpus,
}:

let
  ovmf-snp = "${OVMF-SNP}/FV/OVMF.fd";
  kernel = "${os-image}/bzImage";
  initrd = "${os-image}/initrd";

  # Kata uses a base command line and then appends the command line from the kata config (i.e. also our node-installer config).
  # Thus, we need to perform the same steps when calculating the digest.
  baseCmdline = if debug then kata.runtime.cmdline.debug else kata.runtime.cmdline.default;
  cmdline = lib.strings.concatStringsSep " " [
    baseCmdline
    os-image.cmdline
  ];

in

stdenvNoCC.mkDerivation {
  name = "snp-launch-digest";
  inherit (os-image) version;

  dontUnpack = true;

  buildPhase = ''
    mkdir $out
    ${lib.getExe python3Packages.sev-snp-measure} \
      --mode snp \
      --ovmf ${ovmf-snp} \
      --vcpus ${toString vcpus} \
      --vcpu-type EPYC-Milan \
      --kernel ${kernel} \
      --initrd ${initrd} \
      --append "${cmdline}" \
      --output-format hex > $out/milan.hex
    ${lib.getExe python3Packages.sev-snp-measure} \
      --mode snp \
      --ovmf ${ovmf-snp} \
      --vcpus ${toString vcpus} \
      --vcpu-type EPYC-Genoa \
      --kernel ${kernel} \
      --initrd ${initrd} \
      --append "${cmdline}" \
      --output-format hex > $out/genoa.hex

    # cut newlines
    for file in $out/*.hex; do
      truncate -s -1 "$file"
    done
  '';
}
