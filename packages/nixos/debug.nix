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
      gnugrep
    ];

    services.getty.autologinUser = "root";

    boot.kernelParams = [ "console=ttyS0" ];
    boot.initrd.systemd.emergencyAccess = true;
    systemd.enableEmergencyMode = true;

    # boot.kernelModules = lib.mkForce [];
    # boot.initrd.kernelModules = lib.mkForce [];

    # users.users.root.hashedPassword = "";
    # users.users.root.hashedPassword = "$y$j9T$Reh8JLZxWs32RCjgWstcp1$I4PucGQuG9n/zzQcTC/qcU93BcuPZmI7QUB1ac98ZKB";

    # users.users.root.hashedPassword = ""; # "" means passwordless login
    # services.openssh.settings.PermitRootLogin = "yes";
    # services.openssh.settings.PermitEmptyPasswords = "yes";
    # security.pam.services.sshd.allowNullPassword = true;
  };
}
