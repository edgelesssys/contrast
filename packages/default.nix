# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, pkgs }:

let
  pkgs' = pkgs // self;
  callPackage = lib.callPackageWith pkgs';
  callPackages = lib.callPackagesWith pkgs';
  self' = lib.packagesFromDirectoryRecursive {
    callPackage = lib.callPackageWith pkgs';
    directory = ./by-name;
  };
  self = self' // {
    containers = callPackages ./containers.nix { pkgs = pkgs'; };
    scripts = callPackages ./scripts.nix { pkgs = pkgs'; };
    contrast-releases = callPackages ./contrast-releases.nix { };
    mkNixosConfig = callPackage ./by-name/mkNixosConfig/package.nix { pkgs = pkgs'; };
    microsoft = self'.microsoft // {
      genpolicy = pkgs.pkgsStatic.callPackage ./by-name/microsoft/genpolicy/package.nix { };
      cloud-hypervisor = pkgs.pkgsStatic.callPackage ./by-name/microsoft/cloud-hypervisor/package.nix { };
    };
    kata = self'.kata // {
      genpolicy = pkgs.pkgsStatic.callPackage ./by-name/kata/genpolicy/package.nix {
        inherit (self) kata; # This is only to inherit src/version, must not be depended on.
      };
      kata-runtime = pkgs.pkgsStatic.callPackage ./by-name/kata/kata-runtime/package.nix { };
    };
    qemu-static = pkgs.pkgsStatic.callPackage ./by-name/qemu-static/package.nix { };
    qemu-tdx-static = pkgs.pkgsStatic.callPackage ./by-name/qemu-tdx-static/package.nix { };
  };
in
self
