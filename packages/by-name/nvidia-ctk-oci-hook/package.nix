# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  writeShellApplication,
  nvidia-ctk-with-config,
  lib,
}:
writeShellApplication {
  name = "nvidia-ctk-oci-hook";

  text = ''
    # Log the o/p of the hook to a file
    ${lib.getExe' nvidia-ctk-with-config.tools "nvidia-container-runtime-hook"} \
      -config ${nvidia-ctk-with-config}/etc/nvidia-container-runtime/config.toml \
      -debug "$@" > /var/log/nvidia-hook.log 2>&1
  '';

  meta = {
    description = "OCI hook for nvidia-container-runtime";
    longDescription = ''
      This is an OCI hook (prestart) for the nvidia-container-runtime. It is used to
      facilitate GPU containers in peer pods with the necessary drivers, libraries,
      and binaries to access the GPU.
    '';
  };
}
