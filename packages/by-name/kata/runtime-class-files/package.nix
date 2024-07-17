# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  stdenvNoCC,
  kata,
  OVMF-SNP,
  OVMF,
  debugRuntime ? false,
  qemu-static,
  fetchzip,
}:

let
  image = kata.kata-image;
  kernel = "${kata.kata-kernel-uvm}/bzImage";

  qemu-snp = {
    bin = "${qemu-static}/bin/qemu-system-x86_64";
    share = "${qemu-static}/share/qemu";
  };

  ovmf-snp = "${OVMF-SNP}/FV/OVMF.fd";

  # TODO(msanft): Incorporate the Canonical TDX QEMU patches in our QEMU build for a dynamically
  # built SEV / TDX QEMU binary. For now, take the blob from a build of the following, which matches
  # what Canonical provides in Ubuntu 24.04.
  # https://code.launchpad.net/~kobuk-team/+recipe/tdx-qemu-noble
  qemu-tdx =
    let
      qemu-tdx-blob = fetchzip {
        url = "https://cdn.confidential.cloud/contrast/node-components/1%3A8.2.2%2Bds-0ubuntu2%2Btdx1.0~tdx1.202407031834~ubuntu24.04.1/1%3A8.2.2%2Bds-0ubuntu2%2Btdx1.0~tdx1.202407031834~ubuntu24.04.1.zip";
        hash = "sha256-6TztmmmO2N1jk/cNKdvd/MMIf43N7lxPaasjKARRVik=";
      };
    in
    {
      bin = "${qemu-tdx-blob}/bin/qemu-system-x86_64";
      share = "${qemu-tdx-blob}/share/qemu";
    };

  ovmf-tdx = "${OVMF.fd}/FV/OVMF.fd";

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
    sha256sum ${image} ${kernel} ${qemu-snp.bin} ${qemu-tdx.bin} ${containerd-shim-contrast-cc-v2} ${ovmf-snp} ${ovmf-tdx} | sha256sum | cut -d " " -f 1 > $out/launch-digest.hex
    printf "contrast-cc-%s" "$(cat $out/launch-digest.hex | head -c 32)" > $out/runtime-handler
  '';

  passthru = {
    inherit
      kernel
      image
      qemu-tdx
      qemu-snp
      containerd-shim-contrast-cc-v2
      ovmf-tdx
      ovmf-snp
      kata-runtime
      debugRuntime
      ;
  };
}
