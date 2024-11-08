# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  buildVerityUKI,
  mkNixosConfig,

  withDebug ? true,
}:

let
  config = mkNixosConfig {
    contrast = {
      debug.enable = withDebug;
      gpu.enable = false;
      azure.enable = false;
    };

    environment.etc."machine-id" = {
      mode = "0444";
      text = "0d09941d9cc3b28c04c731736bb12296";
    };

    boot.initrd.enable = true;
    # boot.loader.initScript.enable = lib.mkForce true;
  };
in
{
  image = buildVerityUKI config;
  inherit config;
}
