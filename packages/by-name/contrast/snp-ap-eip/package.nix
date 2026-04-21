# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  stdenvNoCC,
  kata,
  OVMF-SNP,
}:
let
  ovmf-snp = "${OVMF-SNP}/FV/OVMF.fd";
in
stdenvNoCC.mkDerivation {
  name = "snp-ap-eip";

  dontUnpack = true;

  buildPhase = ''
    mkdir $out
    ${lib.getExe kata.sev-snp-measure} ap-eip \
      --ovmf ${ovmf-snp} > $out/ap-eip.hex
    for file in $out/*.hex; do
      truncate -s -1 "$file"
    done
  '';
}
