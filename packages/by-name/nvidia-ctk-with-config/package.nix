# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  nvidia-container-toolkit,
  libnvidia-container,
  replaceVars,
  glibc,
  lib,
}:
nvidia-container-toolkit.override {
  configTemplatePath = (
    replaceVars ./config.toml {
      "nvidia-container-cli" = "${lib.getExe' libnvidia-container "nvidia-container-cli"}";
      "nvidia-container-runtime-hook" = "${lib.getExe' nvidia-container-toolkit "nvidia-container-runtime-hook"}";
      "nvidia-ctk" = "${lib.getExe' nvidia-container-toolkit "nvidia-ctk"}";
      "glibcbin" = "${lib.getBin glibc}";
    }
  );
}