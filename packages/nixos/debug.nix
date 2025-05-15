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
      busybox
      tpm2-tools
      ncurses
      findutils
      curlMinimal
      util-linux
      coreutils
      strace
      tdx-tools
    ];

    services.getty.autologinUser = "root";

    boot.initrd.systemd.emergencyAccess = true;
    systemd.enableEmergencyMode = true;
  };
}
