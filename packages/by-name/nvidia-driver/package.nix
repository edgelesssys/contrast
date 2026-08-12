# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  callPackage,
  kata,
  linuxPackagesFor,
}:

callPackage ./driver.nix {
  inherit (linuxPackagesFor kata.kernel-uvm-gpu)
    nvidiaPackages
    ;
}
