# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  callPackage,
  kata,
  linuxPackagesFor,
}:

# The kernel modules have to be built against the very kernel the PodVM boots, so this
# must stay in sync with the `boot.kernelPackages` set in packages/nixos/kata.nix.
callPackage ./driver.nix {
  inherit (linuxPackagesFor (kata.kernel-uvm.override { withGPU = true; }))
    nvidiaPackages
    ;
}
