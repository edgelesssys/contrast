# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  nixos,
  pkgs,
  jq,

  withDebug ? true,
  withGPU ? true,
  withCSP ? "azure",
}:

let
  # We write this placeholder into the command line and replace it with the real root hash
  # after the image is built.
  roothashPlaceholder = "61fe0f0c98eff2a595dd2f63a5e481a0a25387261fa9e34c37e3a4910edf32b8";

  # 'nixos' uses 'pkgs' from the point in time where nixpkgs function is evaluated. According
  # to the documentation, we should be able to overwrite 'pkgs' by setting nixpkgs.pkgs in
  # the config, but that doesn't seem to work. We use an overlay for now instead.
  # TODO(katexochen): Investigate why the config option doesn't work.
  outerPkgs = pkgs;
in

(nixos (
  { modulesPath, ... }:

  {
    imports = [
      "${modulesPath}/image/repart.nix"
      "${modulesPath}/system/boot/uki.nix"
      ./azure.nix
      ./debug.nix
      ./gpu.nix
      ./image.nix
      ./system.nix
    ];

    contrast.debug.enable = withDebug;
    contrast.gpu.enable = withGPU;
    contrast.azure.enable = withCSP == "azure";

    # TODO(katexochen): imporve, see comment above.
    nixpkgs.overlays = [
      (_self: _super: {
        inherit (outerPkgs) azure-no-agent kernel-podvm-azure;
        inherit (outerPkgs.kata) kata-agent;
      })
    ];

    boot.kernelParams = [ "roothash=${roothashPlaceholder}" ];
  }
)).image.overrideAttrs
  (oldAttrs: {
    nativeBuildInputs = oldAttrs.nativeBuildInputs ++ [ jq ];
    # Replace the placeholder with the real root hash.
    # Only replace first occurrence, or integrity of erofs will be compromised.
    postInstall = ''
      realRoothash=$(${lib.getExe jq} -r "[.[] | select(.roothash != null)] | .[0].roothash" $out/repart-output.json)
      sed -i "0,/${roothashPlaceholder}/ s/${roothashPlaceholder}/$realRoothash/" $out/${oldAttrs.pname}_${oldAttrs.version}.raw
    '';
  })
