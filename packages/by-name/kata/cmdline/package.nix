# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  kata,
  contrastPkgs,
  jq,
  runCommand,
}:

let
  inherit (contrastPkgs.contrast.node-installer-image) os-image withDebug;

  make =
    { os-image, withDebug }:
    runCommand "cmdline" { } ''
      echo -n > "$out" \
        "${lib.concatStringsSep " " (kata.runtime.cmdline.prefix withDebug)}" \
        "$(cat "${os-image}/cmdline")"
      suffix="${lib.concatStringsSep " " (kata.runtime.cmdline.suffix withDebug)}"
      if [[ -n $suffix ]]; then
        echo -n " $suffix" >>"$out"
      fi
    '';

  cmdline = make { inherit os-image withDebug; };
  cmdlineGPU = make {
    inherit (contrastPkgs.contrast.node-installer-image.gpu) os-image;
    inherit withDebug;
  };
in

(runCommand "cmdlines"
  {
    nativeBuildInputs = [ jq ];
  }
  ''
    jq -n >"$out" \
      --rawfile gpu ${cmdlineGPU} \
      --rawfile nogpu ${cmdline} \
      '{ GPU: $gpu, noGPU: $nogpu }'
  ''
)
// {
  inherit make;
}
