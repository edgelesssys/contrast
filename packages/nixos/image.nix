# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  ...
}:

{
  options.contrast.image = {
    microVM = lib.mkEnableOption "Build a micro VM image";
  };

  config = {
    system.image.version = "1-rc1";

    documentation = {
      man.enable = false;
      nixos.enable = false;
    };
  };
}
