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

        contrast =
          let
            runtimeContrast = runtimePkgs.contrastPkgs.contrast;

            nativeContrast = {
              cli = self.callPackage ../packages/by-name/contrast/cli/package.nix {
                inherit (self.contrast) reference-values;
              };
              cli-release = self.callPackage ../packages/by-name/contrast/cli-release/package.nix {
                inherit (self.contrast) cli;
              };
              resourcegen = self.callPackage ../packages/by-name/contrast/resourcegen/package.nix {
                inherit (self.contrast) reference-values;
              };
              contrast = self.callPackage ../packages/by-name/contrast/contrast/package.nix {
                inherit (self.contrast) reference-values;
              };
              e2e = self.callPackage ../packages/by-name/contrast/e2e/package.nix {
                inherit (self.contrast) contrast;
              };
            };
          in
          runtimeContrast // nativeContrast;

        inherit (runtimePkgs.contrastPkgs) k8s-log-collector;
        inherit (runtimePkgs.contrastPkgs) boot-image;
        inherit (runtimePkgs.contrastPkgs) boot-microvm;
        inherit (runtimePkgs.contrastPkgs) qemu-cc;
        inherit (runtimePkgs.contrastPkgs) pause-bundle;
      }
    );
  }
