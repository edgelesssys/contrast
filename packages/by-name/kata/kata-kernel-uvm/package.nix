# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ fetchurl
, linuxManualConfig
}:

linuxManualConfig rec {
  version = "6.7";

  # See https://github.com/kata-containers/kata-containers/blob/5f11c0f144037d8d8f546c89a0392dcd84fa99e2/versions.yaml#L198-L201
  src = fetchurl {
    url = "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-${version}.tar.xz";
    sha256 = "sha256-7zEUSiV20IDYwxaY6D7J9mv5fGd/oqrw1bu58zRbEGk=";
  };

  # Built from Kata upstream via:
  # kata-containers/tools/packaging/kernel/build-kernel.sh -a x86_64 -m -t kvm -x -f setup
  # Note: initramfs sources are removed, as we do not supply them in the build.
  # TODO(msanft): possibly package the config generation in Nix.
  configfile = ./kernel.config;

  allowImportFromDerivation = true;
}
