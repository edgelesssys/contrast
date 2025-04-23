# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

# This builds an nvidia-container-toolkit package with a custom config required
# for use in peer pods GPU containers.

{
  nvidia-container-toolkit,
  libnvidia-container,
  replaceVars,
  glibc,
  lib,
}:
nvidia-container-toolkit.override {
  configTemplatePath = replaceVars ./config.toml {
    "nvidia-container-cli" = "${lib.getExe' libnvidia-container "nvidia-container-cli"}";
    "nvidia-container-runtime-hook" =
      "${lib.getExe' nvidia-container-toolkit.tools "nvidia-container-runtime-hook"}";
    "nvidia-ctk" = "${lib.getExe' nvidia-container-toolkit "nvidia-ctk"}";
    "glibcbin" = "${lib.getBin glibc}";
  };
}
