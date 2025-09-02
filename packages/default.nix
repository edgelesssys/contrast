# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ pkgs }:

let
  inherit (pkgs.lib) makeScope;
in

makeScope pkgs.newScope (
  self:
  pkgs.lib.packagesFromDirectoryRecursive {
    inherit (self) callPackage newScope;
    directory = ./by-name;
  }
  // {
    contrastPkgsStatic = makeScope pkgs.pkgsStatic.newScope (
      self:
      pkgs.lib.packagesFromDirectoryRecursive {
        inherit (self) callPackage newScope;
        directory = ./by-name;
      }
    );
    scripts = pkgs.callPackages ./scripts.nix { };
    containers = pkgs.callPackages ./containers.nix { };
    contrast-releases = pkgs.callPackages ./contrast-releases.nix { };
  }
)
