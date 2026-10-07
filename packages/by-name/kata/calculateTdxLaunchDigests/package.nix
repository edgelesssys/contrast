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
  withDebug ? false,
  maxVcpus ? 1,
}:

let
  ovmf-tdx = "${ovmf}/FV/OVMF.fd";
  kernel = "${os-image}/bzImage";
  initrd = "${os-image}/initrd";
  cmdline = kata.cmdline.make { inherit os-image withDebug; };
  # Hardcode this to the B200 for now, since the calculator only
  # distinguishes between GPU and non-GPU.
  gpuFlag = lib.optionalString withGPU "-g b200";
  # withDebug enables Kata's legacy serial topology, which changes ACPI.
  # GPU VMs currently measure no ACPI tables.
  acpiBlobs = kata.qemuACPIBlobs {
    legacySerial = withDebug;
    inherit maxVcpus;
  };
  acpiBlobsFlag = lib.optionalString (!withGPU) "--acpi-blobs ${acpiBlobs}/$vcpus";
in

stdenvNoCC.mkDerivation {
  name = "tdx-launch-digests";
  inherit (os-image) version;

  dontUnpack = true;

  buildPhase = ''
    mkdir -p $out/rtmr0

    cmdline=$(cat "${cmdline}")
    for vcpus in $(seq 1 ${toString maxVcpus}); do
      ${lib.getExe tdx-measure} rtmr ${gpuFlag} ${acpiBlobsFlag} -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c "$cmdline" 0 > $out/rtmr0/$vcpus.hex
    done

    ${lib.getExe tdx-measure} mrtd -f ${ovmf-tdx} --eventlog-dir eventlogs > $out/mrtd.hex
    ${lib.getExe tdx-measure} rtmr ${gpuFlag} -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c "$cmdline" 1 > $out/rtmr1.hex
    ${lib.getExe tdx-measure} rtmr ${gpuFlag} -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c "$cmdline" 2 > $out/rtmr2.hex
    ${lib.getExe tdx-measure} rtmr ${gpuFlag} -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c "$cmdline" 3 > $out/rtmr3.hex

    cp -r eventlogs $out/
    echo "Eventlog available in $out/eventlogs/"
  '';
}
