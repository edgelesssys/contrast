# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, jq }:

let
  roothashPlaceholder = "61fe0f0c98eff2a595dd2f63a5e481a0a25387261fa9e34c37e3a4910edf32b8";
in

nixos-config:

(nixos-config.override {
  # Inject the `roothash` parameter into the kernel command line,
  # using a placeholder for the verity root hash.
  boot.kernelParams = lib.optional (roothashPlaceholder != "") "roothash=${roothashPlaceholder}";
}).image.overrideAttrs
  (oldAttrs: {
    nativeBuildInputs = oldAttrs.nativeBuildInputs ++ [ jq ];
    # Replace the placeholder with the real root hash.
    # The real root hash is only known after we build the image, so this
    # is injected into the derivation that builds the image.
    # Only replace first occurrence, or integrity of erofs will be compromised.
    postInstall = ''
      realRoothash=$(${lib.getExe jq} -r "[.[] | select(.roothash != null)] | .[0].roothash" $out/repart-output.json)
      sed -i "0,/${roothashPlaceholder}/ s/${roothashPlaceholder}/$realRoothash/" $out/${oldAttrs.pname}_${oldAttrs.version}.raw
    '';
    dontPatchELF = true;
  })
