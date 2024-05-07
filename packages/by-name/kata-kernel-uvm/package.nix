# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ fetchurl
, linuxManualConfig
, patchutils
, kata-agent
}:
let
  kver = "6.1.0";
  modDirVersion = "${kver}.mshv16";
  tarfs_make = ./src;
  tarfs_patch = fetchurl {
    name = "tarfs.patch";
    # update whenever tarfs.c changes: https://github.com/microsoft/kata-containers/commits/msft-main/src/tarfs/tarfs.c
    url = "https://raw.githubusercontent.com/microsoft/kata-containers/${kata-agent.version}/src/tarfs/tarfs.c";
    hash = "sha256-3vuwCOZHgmy0tV9tcgpIRjLxXa4EwNuWIbt9UkRUcDE=";
    downloadToTemp = true;
    recursiveHash = true;
    nativeBuildInputs = [ tarfs_make patchutils ];
    # create a diff where files under fs/tarfs are added to the kernel build
    # "a" is the kernel source tree without tarfs
    # "b" is the kernel source tree with tarfs
    postFetch = ''
      mkdir -p /build/a
      install -D $downloadedFile /build/b/fs/tarfs/tarfs.c
      cp -rT ${tarfs_make} /build/b
      cd /build && diff -Naur a b > /build/tarfs.patch || true
      # remove timestamps
      filterdiff --remove-timestamps /build/tarfs.patch > $out
    '';
  };
in
linuxManualConfig {
  src = fetchurl {
    # Kernel source as defined in
    # https://github.com/microsoft/azurelinux/blob/59ce246f224f282b3e199d9a2dacaa8011b75a06/SPECS/kernel-uvm/kernel-uvm.spec#L19
    url = "https://cblmarinerstorage.blob.core.windows.net/sources/core/kernel-uvm-${modDirVersion}.tar.gz";
    hash = "sha256-8EU8NmU4eiqHdDeCNH28y2wKLaHx6fNcBKzWupqf2Sw=";
  };
  kernelPatches = [
    # this patches the existing Makefile and Kconfig to know about CONFIG_TARFS_FS and fs/tarfs
    { name = "build_tarfs"; patch = ./0001-kernel-uvm-6-1-build-tarfs.patch; }
    # this adds fs/tarfs
    { name = "tarfs"; patch = tarfs_patch; }
  ];
  configfile = fetchurl {
    url = "https://raw.githubusercontent.com/microsoft/azurelinux/59ce246f224f282b3e199d9a2dacaa8011b75a06/SPECS/kernel-uvm/config";
    # Contrast additionally requires the following features:
    # - erofs
    #
    # Contrast uses erofs instead of ext4 (which is used by the AKS runtime),
    # because it is optimized for read-only workloads (speed, image size) and it
    # is trivial to generate reproducible erofs images from a tar file.
    postFetch = ''
      cat <<- EOF >> $out
      CONFIG_MISC_FILESYSTEMS=y
      CONFIG_EROFS_FS=y
      CONFIG_EROFS_FS_XATTR=y
      CONFIG_EROFS_FS_POSIX_ACL=y
      CONFIG_EROFS_FS_SECURITY=y
      CONFIG_EROFS_FS_ZIP=y
      CONFIG_EROFS_FS_ONDEMAND=y
      EOF
    '';
    hash = "sha256-c1+FQzzJQbAvRhV2j0OqRYWcET5kMqvz3vNL7exkudg=";
  };
  version = kver;
  inherit modDirVersion;
  # Allow reading the kernel config
  # this is required to allow nix
  # evaluation to depend on cfg
  # and correctly build everything.
  # Without this, the kernel build
  # has no support for modules.
  allowImportFromDerivation = true;
}
