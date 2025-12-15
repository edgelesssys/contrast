# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  stdenvNoCC,
  kata,
  tdx-measure,
}:

{
  os-image,
  ovmf,
  withGPU ? false,
  debug,
}:

let
  ovmf-tdx = "${ovmf}/FV/OVMF.fd";
  kernel = "${os-image}/bzImage";
  initrd = "${os-image}/initrd";
  # Kata uses a base command line and then appends the command line from the kata config (i.e. also our node-installer config).
  # Thus, we need to perform the same steps when calculating the digest.
  baseCmdline = if debug then kata.runtime.cmdline.debug else kata.runtime.cmdline.default;
  cmdline = lib.strings.concatStringsSep " " [
    baseCmdline
    os-image.cmdline
  ];
  # Hardcode this to the B200 for now, since we only have a testing system with this GPU.
  # When we get more heterogenous test systems, or when TDX-GPU goes into production use,
  # this needs to be made configurable.
  gpuFlag = if withGPU then "-g b200" else "";
in

stdenvNoCC.mkDerivation {
  name = "tdx-launch-digests";
  inherit (os-image) version;

  dontUnpack = true;

  buildPhase = ''
    mkdir $out

    ${lib.getExe tdx-measure} mrtd -f ${ovmf-tdx} --eventlog-dir eventlogs > $out/mrtd.hex
    ${lib.getExe tdx-measure} rtmr ${gpuFlag} -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c '${cmdline}' 0 > $out/rtmr0.hex
    ${lib.getExe tdx-measure} rtmr ${gpuFlag} -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c '${cmdline}' 1 > $out/rtmr1.hex
    ${lib.getExe tdx-measure} rtmr ${gpuFlag} -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c '${cmdline}' 2 > $out/rtmr2.hex
    ${lib.getExe tdx-measure} rtmr ${gpuFlag} -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c '${cmdline}' 3 > $out/rtmr3.hex

    cp -r eventlogs $out/
    echo "Eventlog available in $out/eventlogs/"
  '';
}
