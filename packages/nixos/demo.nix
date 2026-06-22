# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.contrast.demo;
in

{
  options.contrast.demo = {
    enable = lib.mkEnableOption "Enable the demo image";
  };

  config = lib.mkIf cfg.enable {
    boot.initrd.systemd.enable = true;

    boot.kernelParams = [
      "console=ttyS0"
    ];

    boot.initrd.systemd.storePaths = [
      pkgs.contrastPkgs.contrast.initializer
    ];
    boot.initrd.systemd.services.hello = {
      description = "Hello from initrd";

      wantedBy = [ "initrd.target" ];

      serviceConfig = {
        Type = "oneshot";
        ExecStart = "${pkgs.contrastPkgs.contrast.initializer}/bin/initializer report";
        StandardOutput = "journal+console";
        StandardError = "inherit";
        RemainAfterExit = "yes";
        ExecStop = "${config.systemd.package}/bin/systemctl --force poweroff";
        FailureAction = "poweroff";
      };
    };

    boot.initrd.systemd.services.poweroff = {
      description = "Power off after hello";

      wantedBy = [ "initrd.target" ];
      after = [ "hello.service" ];

      serviceConfig = {
        Type = "oneshot";
        ExecStart = "${pkgs.systemd}/bin/systemctl --force --force poweroff";
        # ExecStart = "${pkgs.systemd}/bin/systemctl status hello.service";
        StandardOutput = "journal+console";
        StandardError = "inherit";
      };
    };

    fileSystems = lib.mkForce {
      "/" = {
      device = "none";
      fsType = "tmpfs";
      };
    };
  };
}
