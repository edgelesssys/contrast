# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  config,
  pkgs,
  lib,
  ...
}:

let
  cfg = config.contrast.qemu;
in

{
  options.contrast.qemu = {
    enable = lib.mkEnableOption "Enable QEMU (bare-metal) specific settings";
  };

  config = lib.mkIf cfg.enable {
    boot.kernelPackages = pkgs.recurseIntoAttrs (pkgs.linuxPackagesFor pkgs.kata-kernel-uvm);

    boot.initrd.systemd.tpm2.enable = lib.mkForce false;
    boot.initrd.systemd.enable = lib.mkForce false;
    boot.initrd.availableKernelModules = lib.mkForce [ ];
  };
}
