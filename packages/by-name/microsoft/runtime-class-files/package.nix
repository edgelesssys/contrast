# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  stdenvNoCC,
  microsoft,
  igvmmeasure,
  debugRuntime ? false,
}:

let
  igvm = if debugRuntime then microsoft.kata-igvm.debug else microsoft.kata-igvm;
  cloud-hypervisor-exe = lib.getExe microsoft.cloud-hypervisor;
  containerd-shim-contrast-cc-v2 = lib.getExe microsoft.kata-runtime;
in

stdenvNoCC.mkDerivation {
  name = "runtime-class-files";
  inherit (microsoft.kata-igvm) version;

  dontUnpack = true;

  buildInputs = [ igvmmeasure ];

  buildPhase = ''
    mkdir -p $out
    igvmmeasure -b ${igvm} | dd conv=lcase > $out/launch-digest.hex
    sha256sum ${igvm} ${cloud-hypervisor-exe} ${containerd-shim-contrast-cc-v2}| cut -d " " -f 1 > $out/runtime-hash.hex
  '';

  passthru = {
    inherit
      debugRuntime
      igvm
      cloud-hypervisor-exe
      containerd-shim-contrast-cc-v2
      ;
    rootfs = microsoft.kata-image;
  };
}
