# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib, pkgs }:

let
  pkgs' = pkgs // self;
  callPackages = lib.callPackagesWith pkgs';
  self' = lib.packagesFromDirectoryRecursive {
    callPackage = lib.callPackageWith pkgs';
    directory = ./by-name;
  };
  self = self' // {
    containers = callPackages ./containers.nix { pkgs = pkgs'; };
    scripts = callPackages ./scripts.nix { pkgs = pkgs'; };
    contrast-releases = callPackages ./contrast-releases.nix { };
    microsoft = self'.microsoft // {
      genpolicy = pkgs.pkgsStatic.callPackage ./by-name/microsoft/genpolicy/package.nix { };
    };
    kata = self'.kata // {
      genpolicy = pkgs.pkgsStatic.callPackage ./by-name/kata/genpolicy/package.nix { };
    };
  };
in
self
