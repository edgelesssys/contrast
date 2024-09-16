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
    ${lib.getExe tdx-measure} rtmr -f ${OVMF-TDX}/FV/OVMF.fd -k ${kata.kata-kernel-uvm}/bzImage 0 > $out/rtmr0.hex
    ${lib.getExe tdx-measure} rtmr -f ${OVMF-TDX}/FV/OVMF.fd -k ${kata.kata-kernel-uvm}/bzImage 1 > $out/rtmr1.hex
    ${lib.getExe tdx-measure} rtmr -f ${OVMF-TDX}/FV/OVMF.fd -k ${kata.kata-kernel-uvm}/bzImage 2 > $out/rtmr2.hex
    ${lib.getExe tdx-measure} rtmr -f ${OVMF-TDX}/FV/OVMF.fd -k ${kata.kata-kernel-uvm}/bzImage 3 > $out/rtmr3.hex
  '';
}
