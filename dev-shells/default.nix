# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  contrast-releases, # TODO(katexochen): nix: remove when package available from overlay
  callPackage,
  callPackages,
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
    }
    // callPackages ./contrast-releases.nix {
      inherit contrast-releases; # TODO(katexochen): nix: remove when package available from overlay
    };
in

self
