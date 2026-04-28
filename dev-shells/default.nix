# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  callPackage,
  callPackages,
  git-hooks-lib,
}:

let
  # Note: self must be a flat attribute set, as is expected by nix flakes.
  self =
    lib.packagesFromDirectoryRecursive {
      inherit callPackage;
      directory = ./by-name;
    }
    // {
      default = self.development;
      development = callPackage ./by-name/development.nix { inherit git-hooks-lib; };
    }
    // callPackages ./contrast-releases.nix { };
in

self
