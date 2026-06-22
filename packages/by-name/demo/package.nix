# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  buildVerityMicroVM,
  mkNixosConfig,
}:

buildVerityMicroVM (mkNixosConfig {
  contrast = {
    # debug.enable = withDebug;
    # gpu.enable = withGPU;
    # badaml.enable = withBadAMLTarget;
    # kata.enable = true;
    # kata.guestImagePull = true;
    image.microVM = true;
    demo.enable = true;
  };
})
