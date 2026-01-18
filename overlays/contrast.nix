# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

final: prev:

let
  # On x86_64-linux (inside runtimePkgs), runtimePkgs isn't in scope, use final (itself).
  # On other systems (inside pkgs), runtimePkgs was injected by the flake overlay.
  runtimePkgs = prev.runtimePkgs or final;

  baseContrastPkgs = import ../packages { pkgs = final; };
in
if prev.stdenv.hostPlatform.system == "x86_64-linux" then
  { contrastPkgs = baseContrastPkgs; }
else
  {
    contrastPkgs = baseContrastPkgs.overrideScope (
      cFinal: cPrev: {
        mkNixosConfig = cPrev.mkNixosConfig.override {
          pkgs = runtimePkgs;
        };

        kata = cPrev.kata // {
          # Re-evaluate image to pick up the new mkNixosConfig
          image = cFinal.callPackage ../packages/by-name/kata/image/package.nix { };

          inherit (runtimePkgs.contrastPkgs.kata) contrast-node-installer-image agent kernel-uvm;
        };

        contrast =
          let
            runtimeContrast = runtimePkgs.contrastPkgs.contrast;

            nativeContrast = {
              cli = cFinal.callPackage ../packages/by-name/contrast/cli/package.nix {
                inherit (cFinal.contrast) contrast reference-values;
              };
              cli-release = cFinal.callPackage ../packages/by-name/contrast/cli-release/package.nix {
                inherit (cFinal.contrast) cli;
              };
              resourcegen = cFinal.callPackage ../packages/by-name/contrast/resourcegen/package.nix {
                inherit (cFinal.contrast) contrast reference-values;
              };
              contrast = cFinal.callPackage ../packages/by-name/contrast/contrast/package.nix {
                inherit (cFinal.contrast) reference-values;
              };
              e2e = cFinal.callPackage ../packages/by-name/contrast/e2e/package.nix {
                inherit (cFinal.contrast) contrast;
              };
            };
          in
          runtimeContrast // nativeContrast;

        inherit (runtimePkgs.contrastPkgs)
          debugshell
          tdx-tools
          service-mesh
          k8s-log-collector
          boot-image
          boot-microvm
          qemu-cc
          pause-bundle
          ;
      }
    );
  }
