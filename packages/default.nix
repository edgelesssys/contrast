# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ pkgs }:

let
  inherit (pkgs.lib) makeScope packagesFromDirectoryRecursive;

  # IMPORTANT!
  # byName must be the top-level scope, otherwise overrides won't propagate correctly.
  # Do not merge (//) anything into byName. Always use overrideScope for modifications.
  byName = packagesFromDirectoryRecursive {
    inherit (pkgs) newScope;
    callPackage = pkgs.newScope { };
    directory = ./by-name;
  };
in

byName.overrideScope (
  _final: prev: {
    contrastPkgsStatic = makeScope pkgs.pkgsStatic.newScope (
      self:
      packagesFromDirectoryRecursive {
        inherit (self) callPackage newScope;
        directory = ./by-name;
      }
    );
    scripts = prev.scripts.overrideScope (_: _: pkgs.callPackages ./scripts.nix { });
    containers = pkgs.callPackages ./containers.nix { };
    contrast-releases = pkgs.callPackages ./contrast-releases.nix { };
  }
)
