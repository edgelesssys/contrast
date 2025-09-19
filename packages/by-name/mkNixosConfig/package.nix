# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  nixos,
  pkgs,
}:

let
  outerPkgs = pkgs;

  readModulesDir =
    dir:
    lib.pipe (builtins.readDir dir) [
      (lib.filterAttrs (_filename: type: type == "regular"))
      (lib.mapAttrsToList (filename: _type: "${dir}/${filename}"))
    ];
in

lib.makeOverridable (
  args:
  nixos (
    { modulesPath, ... }:

    {
      imports = [
        "${modulesPath}/image/repart.nix"
        "${modulesPath}/system/boot/uki.nix"
      ]
      ++ readModulesDir ../../nixos;

      nixpkgs.pkgs = outerPkgs;
    }
    // args
  )
)
