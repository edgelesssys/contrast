# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ stdenvNoCC
, microsoft
, igvmmeasure
, debugRuntime ? false
}:

let
  rootfs = microsoft.kata-image;
  igvm = if debugRuntime then microsoft.kata-igvm.debug else microsoft.kata-igvm;
  cloud-hypervisor-bin = "${microsoft.cloud-hypervisor}/bin/cloud-hypervisor";
  containerd-shim-contrast-cc-v2 = "${microsoft.kata-runtime}/bin/containerd-shim-kata-v2";
in

stdenvNoCC.mkDerivation {
  name = "runtime-class-files";
  version = "1714998420";

  dontUnpack = true;

  buildInputs = [ igvmmeasure ];

  buildPhase = ''
    mkdir -p $out
    igvmmeasure -b ${igvm} | dd conv=lcase > $out/launch-digest.hex
    echo -n "contrast-cc-" > $out/runtime-handler
    cat $out/launch-digest.hex | head -c 32 >> $out/runtime-handler
  '';

  passthru = {
    inherit debugRuntime rootfs igvm cloud-hypervisor-bin containerd-shim-contrast-cc-v2;
  };
}
