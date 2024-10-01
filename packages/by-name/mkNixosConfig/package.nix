# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  nixos,
  pkgs,
}:

{
  # If set, placeholder is written into the command line roothash parameter.
  # Used for building verity protected images.
  roothashPlaceholder ? "",

  ...
}@args:

let
  # 'nixos' uses 'pkgs' from the point in time where nixpkgs function is evaluated. According
  # to the documentation, we should be able to overwrite 'pkgs' by setting nixpkgs.pkgs in
  # the config, but that doesn't seem to work. We use an overlay for now instead.
  # TODO(katexochen): Investigate why the config option doesn't work.
  outerPkgs = pkgs;

  readModulesDir =
    dir:
    lib.pipe (builtins.readDir dir) [
      (lib.filterAttrs (_filename: type: type == "regular"))
      (lib.mapAttrsToList (filename: _type: "${dir}/${filename}"))
    ];

  args' = lib.removeAttrs args [ "roothashPlaceholder" ];
in

(nixos (
  { modulesPath, ... }:

  {
    imports = [
      "${modulesPath}/image/repart.nix"
      "${modulesPath}/system/boot/uki.nix"
    ] ++ readModulesDir ../../nixos;

    # TODO(katexochen): imporve, see comment above.
    nixpkgs.overlays = [
      (_self: _super: {
        inherit (outerPkgs)
          azure-no-agent
          cloud-api-adaptor
          kernel-podvm-azure
          pause-bundle
          ;
        inherit (outerPkgs.kata) kata-agent;
      })
    ];

    boot.kernelParams = lib.optional (roothashPlaceholder != "") "roothash=${roothashPlaceholder}";
  }
  // args'
))
