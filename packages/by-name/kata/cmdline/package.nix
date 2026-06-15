# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  kata,
  contrastPkgs,
  writeText,
}:

let
  inherit (contrastPkgs.contrast.node-installer-image) os-image withDebug;

  make =
    { os-image, withDebug }:
    lib.strings.concatStringsSep " " (
      kata.runtime.cmdline.prefix withDebug
      ++ [ os-image.cmdline ]
      ++ kata.runtime.cmdline.suffix withDebug
    );

  cmdline = make { inherit os-image withDebug; };
  cmdlineGPU = make {
    inherit (contrastPkgs.contrast.node-installer-image.gpu) os-image;
    inherit withDebug;
  };
in

(writeText "cmdline" (
  builtins.toJSON {
    GPU = cmdlineGPU;
    noGPU = cmdline;
  }
))
// {
  inherit make;
}
