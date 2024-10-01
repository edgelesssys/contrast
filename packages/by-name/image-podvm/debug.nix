# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.contrast.debug;
in

{
  options.contrast.debug = {
    enable = lib.mkEnableOption "Enable the debugging environment";
  };

  config = lib.mkIf cfg.enable {
    environment.systemPackages = with pkgs; [
      tpm2-tools
      ncurses
      findutils
      curlMinimal
      util-linux
      coreutils
      busybox
      strace
      kata-agent
    ];

    services.getty.autologinUser = "root";
    users.users.root.initialPassword = "";

    boot.kernelParams = [
      "console=ttyS0"
      "console=ttyS1"
      "console=tty0"
      "console=tty1"
    ];

    boot.initrd.systemd.emergencyAccess = true;
    systemd.enableEmergencyMode = true;

    virtualisation.docker = {
        enable = true;
        enableOnBoot = true;

        daemon.settings = {
          features.cdi = true;
        };
    };
  };
}
