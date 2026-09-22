# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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
      # keep-sorted start
      busybox
      contrastPkgs.tdx-tools
      coreutils
      curlMinimal
      findutils
      ncurses
      pciutils
      strace
      tpm2-tools
      util-linux
      # keep-sorted end
    ];

    services.getty.autologinUser = "root";

    boot.initrd.systemd.emergencyAccess = true;
    systemd.enableEmergencyMode = true;
  };
}
