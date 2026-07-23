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
}:

let
  ovmf-tdx = "${ovmf}/FV/OVMF.fd";
  kernel = "${os-image}/bzImage";
  initrd = "${os-image}/initrd";
  cmdline = kata.cmdline.make { inherit os-image withDebug; };
  # Hardcode this to the B200 for now, since we only have a testing system with this GPU.
  # When we get more heterogenous test systems, or when TDX-GPU goes into production use,
  # this needs to be made configurable.
  gpuFlag = lib.optionalString withGPU "-g b200";
  # The nodeinstaller sets use_legacy_serial=true when withDebug is enabled so
  # OVMF's DEBUG_ON_SERIAL_PORT output reaches the host. That drops
  # virtio-serial-pci from the QEMU command line and changes the ACPI tables
  # the firmware measures into RTMR[0]. Tell tdx-measure so it picks the
  # matching set of hardcoded ACPI hashes.
  legacySerialFlag = lib.optionalString withDebug "--legacy-serial";
  # Guest NUMA (enabled for GPU runtime classes) adds one pxb-pcie root bus per guest NUMA node that holds a GPU.
  # We pre-compute RTMR[0] for every possible count 0..maxExtraPciRoots.
  maxExtraPciRoots = if withGPU then 8 else 0;
in

stdenvNoCC.mkDerivation {
  name = "tdx-launch-digests";
  inherit (os-image) version;

  dontUnpack = true;

  buildPhase = ''
    mkdir $out

    ${lib.getExe tdx-measure} mrtd -f ${ovmf-tdx} --eventlog-dir eventlogs > $out/mrtd.hex

    # RTMR[0] depends on the number of extra PCI roots / pxb-pcie bridges, which varies with GPU placement.
    : > $out/rtmr0.hex
    for n in $(seq 0 ${toString maxExtraPciRoots}); do
      ${lib.getExe tdx-measure} rtmr ${gpuFlag} ${legacySerialFlag} --extra-pci-roots "$n" -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c '${cmdline}' 0 >> $out/rtmr0.hex
      echo >> $out/rtmr0.hex
    done
    ${lib.getExe tdx-measure} rtmr ${gpuFlag} -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c '${cmdline}' 1 > $out/rtmr1.hex
    ${lib.getExe tdx-measure} rtmr ${gpuFlag} -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c '${cmdline}' 2 > $out/rtmr2.hex
    ${lib.getExe tdx-measure} rtmr ${gpuFlag} -f ${ovmf-tdx} -k ${kernel} -i ${initrd} -c '${cmdline}' 3 > $out/rtmr3.hex

    cp -r eventlogs $out/
    echo "Eventlog available in $out/eventlogs/"
  '';
}
