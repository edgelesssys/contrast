# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  buildMicroVM,
  mkNixosConfig,

  withDebug ? true,
  withGPU ? false,
  withCSP ? "azure",
}:

buildMicroVM (mkNixosConfig {
  contrast = {
    debug.enable = withDebug;
    gpu.enable = withGPU;
    qemu.enable = true;
  };
})
