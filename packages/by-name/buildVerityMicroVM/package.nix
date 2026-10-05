# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Builds a micro VM image (i.e. rootfs, kernel and kernel cmdline) from a NixOS
# configuration. These components can then be booted in a microVM-fashion
# with QEMU's direct Linux boot feature.
# See: https://qemu-project.gitlab.io/qemu/system/linuxboot.html

{
  symlinkJoin,
  lib,
  jq,
  buildRootFS,
}:

nixos-config:

let
  image = buildRootFS nixos-config.config;
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

    # Calculate the kernel commandline and store it alongside the outputs it depends on.
    nativeBuildInputs = [ jq ];
    postBuild = ''
      roothash=$(jq --exit-status --raw-output '.[0].roothash' <"${image}/repart-output.json")

      echo -n >"$out/cmdline" \
        ${lib.escapeShellArgs nixos-config.config.boot.kernelParams} \
        "init=${nixos-config.config.system.build.toplevel}/init" \
        "roothash=$roothash" \
        "cgroup_no_v1=all"
    '';

    passthru = {
      inherit (nixos-config.config.system.build)
        kernel
        initialRamdisk
        toplevel
        ;
      inherit image;
    };
  }
