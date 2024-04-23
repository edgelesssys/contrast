# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib, pkgs }:

let
  pkgs' = pkgs // self;
  callPackages = lib.callPackagesWith pkgs';
  self = (lib.packagesFromDirectoryRecursive {
    callPackage = lib.callPackageWith pkgs';
    directory = ./by-name;
  }) // {
    containers = callPackages ./containers.nix { pkgs = pkgs'; };
    scripts = callPackages ./scripts.nix { pkgs = pkgs'; };
    genpolicy-msft = pkgs.pkgsStatic.callPackage ./by-name/genpolicy-msft/package.nix { };
    genpolicy-kata = pkgs.pkgsStatic.callPackage ./by-name/genpolicy-kata/package.nix { };
  };
in
self
