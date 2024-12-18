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
      package = lib.mkDefault (
        config.boot.kernelPackages.nvidiaPackages.mkDriver {
          # TODO: Investigate why the latest version breaks
          # GPU containers.
          version = "550.90.07";
          sha256_64bit = "sha256-Uaz1edWpiE9XOh0/Ui5/r6XnhB4iqc7AtLvq4xsLlzM=";
          sha256_aarch64 = "sha256-uJa3auRlMHr8WyacQL2MyyeebqfT7K6VU0qR7LGXFXI=";
          openSha256 = "sha256-VLmh7eH0xhEu/AK+Osb9vtqAFni+lx84P/bo4ZgCqj8=";
          settingsSha256 = "sha256-sX9dHEp9zH9t3RWp727lLCeJLo8QRAGhVb8iN6eX49g=";
          persistencedSha256 = "sha256-qe8e1Nxla7F0U88AbnOZm6cHxo57pnLCqtjdvOvq9jk=";
        }
      );
      nvidiaPersistenced = true;
    };
    hardware.graphics = {
      enable = true;
      enable32Bit = true;
    };
    hardware.nvidia-container-toolkit.enable = true;

    image.repart.partitions."10-root".contents."/usr/share/oci/hooks/prestart/nvidia-container-toolkit.sh".source =
      lib.getExe pkgs.nvidia-ctk-oci-hook;

    boot.initrd.kernelModules = [
      # Extra kernel modules required to talk to the GPU in CC-Mode.
      "ecdsa_generic"
      "ecdh"
    ];

    services.xserver.videoDrivers = [ "nvidia" ];
  };
}
