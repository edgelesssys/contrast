# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ runtimePkgs }:

final: prev:

let
  baseContrastPkgs = import ../packages {
    pkgs = final;
    inherit runtimePkgs;
  };
in

if prev.stdenv.hostPlatform.system == "x86_64-linux" then
  { contrastPkgs = baseContrastPkgs; }
else
  {
    contrastPkgs = baseContrastPkgs.overrideScope (
      self: super: {
        mkNixosConfig = super.mkNixosConfig.override {
          pkgs = runtimePkgs;
        };

        kata = super.kata // {
          # Re-evaluate image to pick up the new mkNixosConfig
          image = self.callPackage ../packages/by-name/kata/image/package.nix { };

          inherit (runtimePkgs.contrastPkgs.kata) contrast-node-installer-image;
          inherit (runtimePkgs.contrastPkgs.kata) agent;
          inherit (runtimePkgs.contrastPkgs.kata) kernel-uvm;
        };
      }
    );
  }
