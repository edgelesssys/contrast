# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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
    kata.guestImagePull = true;
    image.microVM = true;
  };
})
