# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.contrast.gpu;
in

{
  options.contrast.gpu = {
    enable = lib.mkEnableOption "Enable GPU support";
  };

  config = lib.mkIf cfg.enable {
    hardware.nvidia = {
      open = true;
      package = lib.mkDefault config.boot.kernelPackages.nvidiaPackages.production;
      nvidiaPersistenced = true;
    };
    hardware.graphics = {
      enable = true;
      enable32Bit = true;
    };
    hardware.nvidia-container-toolkit.enable = true;

    image.repart.partitions."10-root".contents."/usr/share/oci/hooks/prestart/nvidia-container-toolkit.sh".source = lib.getExe pkgs.nvidia-ctk-oci-hook;

    boot.initrd.kernelModules = [
      # Extra kernel modules required to talk to the GPU in CC-Mode.
      "ecdsa_generic"
      "ecdh"
    ];

    services.xserver.videoDrivers = [ "nvidia" ];
  };
}
