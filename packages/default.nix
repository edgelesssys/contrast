# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ pkgs }:

let
  # IMPORTANT!
  # byName must be the top-level scope, otherwise overrides won't propagate correctly.
  # Do not merge (//) anything into byName. Always use overrideScope for modifications.
  byName = pkgs.lib.packagesFromDirectoryRecursive {
    inherit (pkgs) newScope;
    callPackage = pkgs.newScope { };
    directory = ./by-name;
  };
in

byName.overrideScope (
  _final: prev: {
    contrastPkgsStatic = pkgs.pkgsStatic.contrastPkgs;
    scripts = prev.scripts.overrideScope (_: _: pkgs.callPackages ./scripts.nix { });
    containers = pkgs.callPackages ./containers.nix { };
    contrast-releases = pkgs.callPackages ./contrast-releases.nix { };
  }
)
