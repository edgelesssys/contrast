# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  stdenvNoCC,
  kata,
  fetchzip,
  OVMF,
  debugRuntime ? false,
}:

let
  image = kata.kata-image;
  kernel = "${kata.kata-kernel-uvm}/bzImage";

  # TODO(msanft): building a static qemu with nix.
  qemu-bin =
    let
      qemuDrv = stdenvNoCC.mkDerivation rec {
        pname = "qemu-static-kata";
        version = "3.6.0";

        src = fetchzip {
          url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-static-${version}-amd64.tar.xz";
          hash = "sha256-ynMzMoJ90BzKuE6ih6DmbM2zWTDxsMwkAKsI8pbO3sg=";
        };

        dontBuild = true;

        installPhase = ''
          install -Dt $out/bin kata/bin/qemu-system-x86_64
        '';
      };
    in
    "${qemuDrv}/bin/qemu-system-x86_64";

  ovmf = "${OVMF.fd}/FV/OVMF.fd";

  containerd-shim-contrast-cc-v2 = "${kata.kata-runtime}/bin/containerd-shim-kata-v2";

  kata-runtime = "${kata.kata-runtime}/bin/kata-runtime";
in

stdenvNoCC.mkDerivation {
  name = "runtime-class-files";
  inherit (kata.kata-image) version;

  dontUnpack = true;

  # TODO(msanft): perform the actual launch digest calculation.
  buildPhase = ''
    mkdir -p $out
    sha256sum ${image} ${kernel} ${qemu-bin} ${containerd-shim-contrast-cc-v2} ${ovmf} | sha256sum | cut -d " " -f 1 > $out/launch-digest.hex
    printf "contrast-cc-%s" "$(cat $out/launch-digest.hex | head -c 32)" > $out/runtime-handler
  '';

  passthru = {
    inherit
      kernel
      image
      qemu-bin
      containerd-shim-contrast-cc-v2
      ovmf
      kata-runtime
      debugRuntime
      ;
  };
}
