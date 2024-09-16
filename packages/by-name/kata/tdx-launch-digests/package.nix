# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  stdenvNoCC,
  kata,
  OVMF-TDX,
  tdx-measure,
}:

stdenvNoCC.mkDerivation {
  name = "tdx-launch-digests";
  inherit (kata.kata-image) version;

  dontUnpack = true;

  buildPhase = ''
    mkdir $out

    ${lib.getExe tdx-measure} mrtd -f ${OVMF-TDX}/FV/OVMF.fd > $out/mrtd.hex
  '';
}
