# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  writeShellApplication,
  nvidia-container-toolkit,
  lib,
}:

writeShellApplication {
  name = "nvidia-ctk-prestart";
  runtimeInputs = [ nvidia-container-toolkit ];
  text = ''
    #!/usr/bin/env bash -x

    # Log the o/p of the hook to a file
    ${lib.getExe' nvidia-container-toolkit "nvidia-container-runtime-hook"} -debug "$@" > /var/log/nvidia-hook.log 2>&1
  '';
}
