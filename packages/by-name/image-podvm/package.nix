# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  jq,
  mkNixosConfig,

  withDebug ? true,
  withGPU ? false,
  withCSP ? "azure",
}:

let
  roothashPlaceholder = "61fe0f0c98eff2a595dd2f63a5e481a0a25387261fa9e34c37e3a4910edf32b8";
in

(mkNixosConfig {
  inherit roothashPlaceholder;

  contrast.debug.enable = withDebug;
  contrast.gpu.enable = withGPU;
  contrast.azure.enable = withCSP == "azure";

}).image.overrideAttrs
  (oldAttrs: {
    nativeBuildInputs = oldAttrs.nativeBuildInputs ++ [ jq ];
    # Replace the placeholder with the real root hash.
    # Only replace first occurrence, or integrity of erofs will be compromised.
    postInstall = ''
      realRoothash=$(${lib.getExe jq} -r "[.[] | select(.roothash != null)] | .[0].roothash" $out/repart-output.json)
      sed -i "0,/${roothashPlaceholder}/ s/${roothashPlaceholder}/$realRoothash/" $out/${oldAttrs.pname}_${oldAttrs.version}.raw
    '';
  })
