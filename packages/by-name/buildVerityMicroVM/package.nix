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

let
  image = nixos-config.image.overrideAttrs (oldAttrs: {
    passthru = oldAttrs.passthru // {
      imageFile = "${oldAttrs.pname}_${oldAttrs.version}.raw";
    };
  });
in

lib.throwIf
  (lib.foldlAttrs (
    acc: _: partConfig:
    acc || (partConfig.repartConfig.Type == "esp")
  ) false nixos-config.config.image.repart.partitions)
  "MicroVM images should not contain an ESP."

  symlinkJoin
  {
    pname = "microvm-image";
    inherit (nixos-config.config.system.image) version;

    paths = [
      nixos-config.config.system.build.kernel
      nixos-config.config.system.build.initialRamdisk
      image
    ];

    passthru =
      let
        roothash = builtins.head (
          lib.map (e: e.roothash) (builtins.fromJSON (builtins.readFile "${image}/repart-output.json"))
        );
      in
      {
        cmdline = lib.concatStringsSep " " (
          nixos-config.config.boot.kernelParams
          ++ [
            "init=${nixos-config.config.system.build.toplevel}/init"
            "roothash=${roothash}"
          ]
        );
        inherit (image) imageFile;
      };
  }
