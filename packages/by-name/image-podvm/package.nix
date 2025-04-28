# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  buildVerityUKI,
  mkNixosConfig,

  withDebug ? false,
  withGPU ? false,
  withCSP ? "azure",
}:

buildVerityUKI (mkNixosConfig {
  contrast = {
    debug.enable = withDebug;
    gpu.enable = withGPU;
    azure.enable = withCSP == "azure";
    peerpods.enable = true;
  };
})
