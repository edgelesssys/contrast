# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

# Builds a micro VM image (i.e. rootfs, kernel and kernel cmdline) from a NixOS
# configuration. These components can then be booted in a microVM-fashion
# with QEMU's direct Linux boot feature.
# See: https://qemu-project.gitlab.io/qemu/system/linuxboot.html

{
  symlinkJoin,
  lib,
  ...
}:

nixos-config:

lib.throwIf
  (lib.foldlAttrs (
    acc: _: partConfig:
    acc || (partConfig.repartConfig.Type == "esp")
  ) false nixos-config.config.image.repart.partitions)
  "MicroVM images should not contain an ESP."

  symlinkJoin
  {
    name = "microvm-image";

    paths = [
      nixos-config.config.system.build.kernel
      nixos-config.config.system.build.image
    ];

    postBuild = ''
      echo -n ${lib.concatStringsSep " " nixos-config.config.boot.kernelParams} > $out/kernel-params
    '';
  }
