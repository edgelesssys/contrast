# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  pkgs,
  lib,
  ...
}:

{
  config = {
    systemd.services.imagepuller = {
      description = "Image Puller";
      documentation = [ "https://github.com/edgelesssys/contrast" ];
      wants = [ "network-online.target" ];
      wantedBy = [ "kata-agent.service" ];
      serviceConfig = {
        Type = "exec";
        StandardOutput = "journal+console";
        StandardError = "inherit";
        ExecStart = "${lib.getExe pkgs.imagepuller}";
        Restart = "always";
        LimitNOFILE = 1048576;
      };
    };
  };
}
