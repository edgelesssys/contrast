# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  pkgs,
  runtimePkgs,
}:

let
  inherit (pkgs.lib) makeScope;
in

makeScope pkgs.newScope (
  self:
  let
    fromDir = pkgs.lib.packagesFromDirectoryRecursive {
      inherit (self) callPackage newScope;
      directory = ./by-name;
    };
  in
  fromDir
  // {
    contrastPkgsStatic = makeScope pkgs.pkgsStatic.newScope (
      self:
      pkgs.lib.packagesFromDirectoryRecursive {
        inherit (self) callPackage newScope;
        directory = ./by-name;
      }
    );
    scripts = (fromDir.scripts or { }) // pkgs.callPackages ./scripts.nix { inherit runtimePkgs; };
    containers =
      (fromDir.containers or { }) // pkgs.callPackages ./containers.nix { inherit runtimePkgs; };
    contrast-releases = pkgs.callPackages ./contrast-releases.nix { };
  }
)
