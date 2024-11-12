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

  # lib.throwIf
  # (lib.foldl' (acc: v: acc || (lib.hasInfix "root=" v)) false nixos-config.config.boot.kernelParams)
  # "MicroVM images should not set the `root=` commandline parameter, as it will need to be decided by the VMM."

  symlinkJoin
  {
    name = "microvm-image";

    paths = [
      nixos-config.config.system.build.kernel
      nixos-config.config.system.build.image
    ];

    postBuild =
      let
        kernelParams = nixos-config.config.boot.kernelParams ++ [
          "init=${nixos-config.config.system.build.toplevel}/init"
        ];
      in
      ''
        echo -n ${lib.concatStringsSep " " kernelParams} > $out/kernel-params
      '';
  }
