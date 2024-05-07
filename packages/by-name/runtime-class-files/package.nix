# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ fetchurl
, stdenvNoCC
, igvmmeasure
, kata-image
, kata-igvm
, kata-runtime
, debugRuntime ? false
}:
let
  rootfs = kata-image;
  igvm = if debugRuntime then kata-igvm.debug else kata-igvm;
  cloud-hypervisor-bin = fetchurl {
    url = "https://cdn.confidential.cloud/contrast/node-components/1714998420/cloud-hypervisor-cvm";
    hash = "sha256-coTHzd5/QLjlPQfrp9d2TJTIXKNuANTN7aNmpa8PRXo=";
  };
  containerd-shim-contrast-cc-v2 = "${kata-runtime}/bin/containerd-shim-kata-v2";
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
