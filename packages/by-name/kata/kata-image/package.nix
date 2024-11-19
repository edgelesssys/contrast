# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  buildVerityMicroVM,
  mkNixosConfig,

  withDebug ? false,
  withGPU ? false,
}:

buildVerityMicroVM (mkNixosConfig {
  contrast = {
    debug.enable = withDebug;
    gpu.enable = withGPU;
    kata.enable = true;
    image.microVM = true;
  };
})
