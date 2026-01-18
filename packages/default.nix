# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ pkgs }:

let
  inherit (pkgs.lib) packagesFromDirectoryRecursive;

  # IMPORTANT!
  # byName must be the top-level scope, otherwise overrides won't propagate correctly.
  # Do not merge (//) anything into byName. Always use overrideScope for modifications. inherit (pkgs.lib) packagesFromDirectoryRecursive;
  byName = packagesFromDirectoryRecursive {
    inherit (pkgs) callPackage newScope;
    directory = ./by-name;
  };
in

pkgs.nix-pkgset.lib.makePackageSet "contrastPkgs" pkgs.newScope (
  self:
  byName.overrideScope (
    _final: prev: {
      contrastPkgsStatic = pkgs.pkgsStatic.contrastPkgs;
      scripts = prev.scripts.overrideScope (_: _: pkgs.callPackages ./scripts.nix { });
      containers = pkgs.callPackages ./containers.nix { contrastPkgs = self.contrastPkgsTargetTarget; };
      contrast-releases = pkgs.callPackages ./contrast-releases.nix { };
    }
  )
)
