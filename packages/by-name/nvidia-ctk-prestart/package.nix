# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  writeShellApplication,
  nvidia-ctk-with-config,
  lib,
}:
writeShellApplication {
  name = "nvidia-ctk-prestart";
  text = ''
    # Log the o/p of the hook to a file
    ${lib.getExe' nvidia-ctk-with-config "nvidia-container-runtime-hook"} \
      -config ${nvidia-ctk-with-config}/etc/nvidia-container-runtime/config.toml \
      -debug "$@" > /var/log/nvidia-hook.log 2>&1
  '';
}
