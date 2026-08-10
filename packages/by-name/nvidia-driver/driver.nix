# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  dbus,
  lib,
  nvidiaPackages,
  openssl,
  stdenv,
  zlib,
}:

(
  (nvidiaPackages.mkDriver rec {
    url = "https://us.download.nvidia.com/tesla/${version}/NVIDIA-Linux-x86_64-${version}.run";
    version = "595.71.05";
    sha256_64bit = "sha256-NiA7iWC35JyKQva6H1hjzeNKBek9KyS3mK8G3YRva4I=";
    openSha256 = "sha256-Lfz71QWKM6x/jD2B22SWpUi7/og30HRlXg1kL3EWzEw=";
    # Persistenced release isn't guaranteed to exist for the driver versions we are using, so follow production.
    persistencedVersion = nvidiaPackages.production.persistenced.version;
    persistencedSha256 = nvidiaPackages.production.persistenced.src.outputHash;
    useSettings = false;
  }).override
  {
    disable32Bit = true;
  }
).overrideAttrs
  (_oldAttrs: {
    # We strip the driver package from its dependencies on desktop software like Wayland and X11.
    # For server use-cases, we shouldn't need these. The Mesa (and thus Perl) and libGL dependencies are dropped
    # too, as GPU workloads will likely be AI-related and not graphical. The libdrm dependency is dropped as well,
    # as we're probably not going to be watching Netflix on the servers.
    # Source: https://github.com/NixOS/nixpkgs/blob/eac1633a086e8e109e00ce58c0b47721da1dbdfd/pkgs/os-specific/linux/nvidia-x11/generic.nix#L100C3-L114C6
    libPath = lib.makeLibraryPath [
      zlib
      stdenv.cc.cc
      openssl
      dbus # for nvidia-powerd
    ];
  })
